package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/Zany2/dtoken-go/sso"
)

// deploymentConfig holds one process's settings; secrets must never be logged. deploymentConfig 保存单个进程的配置，禁止记录其中的密钥。
type deploymentConfig struct {
	role, listen, publicURL, centerURL, namespace           string
	redisURL, cookieKey, clientID, clientSecret, signingKey string
	verificationKey                                         string
	cookieVerificationKey                                   string
	demoUser, demoPassword                                  string
	development                                             bool
	clients                                                 map[string]sso.Client
}

var deploymentID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`) // deploymentID keeps cookie names and storage segments unambiguous. deploymentID 保证 Cookie 名称和存储分段无歧义。

// loadConfig validates all local settings before opening storage. loadConfig 在打开存储前校验全部本地配置。
func loadConfig(role string, env func(string) string) (deploymentConfig, error) {
	c := deploymentConfig{role: role, listen: env("SSO_LISTEN"), publicURL: env("SSO_PUBLIC_URL"), centerURL: env("SSO_CENTER_URL"), namespace: env("SSO_NAMESPACE"), redisURL: env("SSO_REDIS_URL"), cookieKey: env("SSO_COOKIE_KEY"), clientID: env("SSO_CLIENT_ID"), clientSecret: env("SSO_CLIENT_SECRET"), signingKey: env("SSO_SIGNING_KEY"), demoUser: env("SSO_DEMO_USER"), demoPassword: env("SSO_DEMO_PASSWORD")}
	c.verificationKey = env("SSO_VERIFICATION_KEY")
	c.cookieVerificationKey = env("SSO_COOKIE_VERIFICATION_KEY")
	if role != "center" && role != "client" {
		return c, errors.New("role must be center or client")
	}
	if value := env("SSO_DEV"); value != "" {
		if value != "true" && value != "false" {
			return c, errors.New("SSO_DEV must be true or false")
		}
		c.development = value == "true"
	}
	if !deploymentID.MatchString(c.namespace) {
		return c, errors.New("SSO_NAMESPACE must contain 1-64 letters, digits, underscores or hyphens")
	}
	if len(c.cookieKey) < 32 {
		return c, errors.New("SSO_COOKIE_KEY requires at least 32 bytes")
	}
	if c.cookieVerificationKey != "" && len(c.cookieVerificationKey) < 32 {
		return c, errors.New("SSO_COOKIE_VERIFICATION_KEY requires at least 32 bytes when set")
	}
	if err := checkOrigin(c.publicURL, c.development); err != nil {
		return c, fmt.Errorf("SSO_PUBLIC_URL: %w", err)
	}
	c.publicURL = canonicalOrigin(c.publicURL)
	host, port, err := net.SplitHostPort(c.listen)
	if err != nil {
		return c, errors.New("SSO_LISTEN must be host:port")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 || (c.development && !loopbackHost(host)) {
		return c, errors.New("SSO_LISTEN has an invalid port or non-loopback development address")
	}
	if !c.development && c.redisURL == "" {
		return c, errors.New("SSO_REDIS_URL is required outside development")
	}
	if role == "client" {
		if !deploymentID.MatchString(c.clientID) {
			return c, errors.New("SSO_CLIENT_ID must contain 1-64 letters, digits, underscores or hyphens")
		}
		if len(c.clientSecret) < 32 || len(c.signingKey) < 32 {
			return c, errors.New("SSO_CLIENT_SECRET and SSO_SIGNING_KEY require at least 32 bytes each")
		}
		if c.verificationKey != "" && len(c.verificationKey) < 32 {
			return c, errors.New("SSO_VERIFICATION_KEY requires at least 32 bytes when set")
		}
		if err := checkOrigin(c.centerURL, c.development); err != nil {
			return c, fmt.Errorf("SSO_CENTER_URL: %w", err)
		}
		c.centerURL = canonicalOrigin(c.centerURL)
		if c.centerURL == c.publicURL {
			return c, errors.New("center and client public origins must differ")
		}
		return c, nil
	}

	// Authentication is deliberately a replaceable demo boundary, never an arbitrary login ID. 认证是需要替换的演示边界，不接受任意登录 ID。
	if env("SSO_ENABLE_DEMO_LOGIN") != "true" || c.demoUser == "" || len(c.demoPassword) < 16 {
		return c, errors.New("explicit SSO_ENABLE_DEMO_LOGIN=true, SSO_DEMO_USER and a 16-byte SSO_DEMO_PASSWORD are required; replace the demo authenticator for production")
	}
	raw := env("SSO_CLIENTS_JSON")
	if len(raw) == 0 || len(raw) > 1<<20 {
		return c, errors.New("SSO_CLIENTS_JSON must be a nonempty JSON array up to 1 MiB")
	}
	var clients []sso.Client
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&clients); err != nil {
		return c, errors.New("SSO_CLIENTS_JSON contains invalid registration JSON")
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF || len(clients) == 0 {
		return c, errors.New("SSO_CLIENTS_JSON must contain exactly one nonempty array")
	}
	c.clients = make(map[string]sso.Client, len(clients))
	for _, client := range clients {
		if client.VerificationKey != "" && len(client.VerificationKey) < 32 {
			return c, errors.New("registration verificationKey requires at least 32 bytes when set")
		}
		if !deploymentID.MatchString(client.ClientID) || len(client.ClientSecret) < 32 || len(client.SecretKey) < 32 {
			return c, errors.New("each registration requires a safe client ID and 32-byte exchange/signing secrets")
		}
		if _, exists := c.clients[client.ClientID]; exists {
			return c, errors.New("SSO_CLIENTS_JSON contains duplicate client IDs")
		}
		if !client.RequirePKCE || len(client.Modes) != 1 || client.Modes[0] != sso.ModeTicket || len(client.RedirectURIs) != 1 {
			return c, errors.New("this example requires Ticket mode, PKCE and exactly one callback per client")
		}
		callback, err := url.Parse(client.RedirectURIs[0])
		if err != nil || callback.User != nil || callback.Path != "/sso/callback" || callback.RawPath != "" || callback.RawQuery != "" || callback.ForceQuery || callback.Fragment != "" || strings.Contains(client.RedirectURIs[0], "#") {
			return c, errors.New("each callback must use the exact /sso/callback path without query or fragment")
		}
		if err := checkOrigin(callback.Scheme+"://"+callback.Host, c.development); err != nil {
			return c, fmt.Errorf("client callback: %w", err)
		}
		origin := canonicalOrigin(callback.Scheme + "://" + callback.Host)
		if origin == c.publicURL {
			return c, errors.New("a client callback must not use the center origin")
		}
		client.RedirectURIs[0] = origin + "/sso/callback"
		c.clients[client.ClientID] = client
	}
	options := sso.HTTPOptions{ServerOptions: sso.ServerOptions{CheckSign: true, Clients: c.clients}, LoginIDResolver: func(*http.Request) (string, bool) { return "", false }}
	if err := options.Validate(); err != nil {
		return c, err
	}
	return c, nil
}

// checkOrigin requires canonical root origins; HTTP is limited to explicit local development. checkOrigin 要求规范的根来源，HTTP 仅限显式本地开发。
func checkOrigin(value string, development bool) error {
	u, err := url.Parse(value)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.ContainsAny(value, "#\\ \t\r\n") || (u.Scheme != "https" && u.Scheme != "http") {
		return errors.New("must be an HTTP(S) origin without path, userinfo, query or fragment")
	}
	if development && !loopbackHost(u.Hostname()) {
		return errors.New("development origins must be loopback hosts")
	}
	if !development && u.Scheme != "https" {
		return errors.New("HTTPS is required outside development")
	}
	if strings.HasSuffix(u.Host, ":") {
		return errors.New("origin port must be nonempty")
	}
	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return errors.New("origin port must be between 1 and 65535")
		}
	}
	return nil
}

// loopbackHost limits development listeners and public URLs to the local machine. loopbackHost 将开发监听和公开地址限制在本机。
func loopbackHost(host string) bool {
	return strings.EqualFold(host, "localhost") || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())
}

// canonicalOrigin serializes a validated origin as browsers do for host case and default ports. canonicalOrigin 按浏览器对主机大小写及默认端口的规则序列化已校验来源。
func canonicalOrigin(value string) string {
	u, _ := url.Parse(value)
	host := strings.ToLower(u.Hostname())
	if ip, err := netip.ParseAddr(host); err == nil {
		host = ip.String()
		// Browsers serialize mapped IPv6 in hexadecimal and keep it distinct from IPv4. 浏览器以十六进制序列化映射 IPv6，且与 IPv4 区分。
		if ip.Is4In6() {
			bytes := ip.As16()
			host = fmt.Sprintf("::ffff:%x:%x", uint16(bytes[12])<<8|uint16(bytes[13]), uint16(bytes[14])<<8|uint16(bytes[15]))
		}
	}
	port := u.Port()
	if port != "" {
		number, _ := strconv.Atoi(port)
		port = strconv.Itoa(number)
	}
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		port = ""
	}
	if port != "" {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return u.Scheme + "://" + host
}

// storagePrefix separates environments, roles, and client instances. storagePrefix 隔离环境、角色和客户端实例。
func (c deploymentConfig) storagePrefix() string {
	if c.role == "center" {
		return "sso-example:" + c.namespace + ":center:"
	}
	return "sso-example:" + c.namespace + ":client:" + c.clientID + ":"
}
