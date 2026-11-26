package acceptance

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// acceptanceClient holds an isolated browser jar and explicit deployment destinations. acceptanceClient 保存独立浏览器 Cookie 容器及显式部署目标。
type acceptanceClient struct {
	http    *http.Client
	origins map[string]bool
}

// acceptanceResponse retains bounded response data in memory only. acceptanceResponse 仅在内存保存受限响应数据。
type acceptanceResponse struct {
	status  int
	header  http.Header
	cookies []*http.Cookie
	body    []byte
}

// acceptanceOrigin accepts HTTPS roots or explicitly local HTTP roots without credentials. acceptanceOrigin 接受不含凭据的 HTTPS 根来源或明确本机 HTTP 根来源。
func acceptanceOrigin(value string) (string, error) {
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.ContainsAny(value, "#\\\r\n\t ") {
		return "", errors.New("expected a root origin without path, credentials, query or fragment")
	}
	if u.Scheme != "https" {
		ip := net.ParseIP(u.Hostname())
		if u.Scheme != "http" || (u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback())) {
			return "", errors.New("HTTP acceptance targets must use loopback addresses")
		}
	}
	return u.String(), nil
}

// newAcceptanceClient keeps TLS verification enabled and disables implicit redirect following. newAcceptanceClient 保持 TLS 证书校验并禁止隐式跟随重定向。
func newAcceptanceClient(origins ...string) (*acceptanceClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	allowed := make(map[string]bool)
	for _, origin := range origins {
		if _, err := acceptanceOrigin(origin); err != nil {
			return nil, err
		}
		allowed[origin] = true
	}
	return &acceptanceClient{http: &http.Client{Jar: jar, Timeout: 15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}, origins: allowed}, nil
}

// target checks a destination before any cookies or credentials can be sent. target 在发送任何 Cookie 或凭据前检查目标。
func (c *acceptanceClient) target(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Fragment != "" || strings.Contains(raw, "#") || !c.origins[u.Scheme+"://"+u.Host] {
		return nil, errors.New("unexpected acceptance destination")
	}
	return u, nil
}

// redirect resolves a single hop and checks its expected application and path. redirect 解析单次跳转并检查预期应用与路径。
func (c *acceptanceClient) redirect(from string, response *acceptanceResponse, origin, path string) (string, error) {
	if response == nil || (response.status != http.StatusFound && response.status != http.StatusSeeOther) || response.header.Get("Location") == "" {
		return "", errors.New("expected a redirect with a location")
	}
	base, err := c.target(from)
	if err != nil {
		return "", err
	}
	next, err := base.Parse(response.header.Get("Location"))
	if err != nil {
		return "", errors.New("invalid redirect location")
	}
	u, err := c.target(next.String())
	if err != nil || u.Scheme+"://"+u.Host != origin || u.Path != path || u.RawPath != "" {
		return "", errors.New("redirect did not match the expected origin and path")
	}
	return next.String(), nil
}

// request reads a bounded response without exposing network diagnostics containing signed URLs. request 读取受限响应，不暴露含签名 URL 的网络诊断。
func (c *acceptanceClient) request(method, target string, form url.Values, origin string, snapshot []*http.Cookie) (*acceptanceResponse, error) {
	if _, err := c.target(target); err != nil {
		return nil, err
	}
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	r, err := http.NewRequest(method, target, body)
	if err != nil {
		return nil, errors.New("cannot construct acceptance request")
	}
	if form != nil {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	client := *c.http
	if snapshot != nil {
		client.Jar = nil
		for _, cookie := range snapshot {
			r.AddCookie(cookie)
		}
	}
	response, err := client.Do(r)
	if err != nil {
		return nil, errors.New("acceptance request failed; check reachability, TLS and timeout")
	}
	defer response.Body.Close()
	const limit = 1 << 20
	payload, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || len(payload) > limit {
		return nil, errors.New("acceptance response read failed or exceeded 1 MiB")
	}
	return &acceptanceResponse{status: response.StatusCode, header: response.Header.Clone(), cookies: response.Cookies(), body: payload}, nil
}

// snapshot copies current cookies for stale-session checks without storing them on disk. snapshot 复制当前 Cookie 用于旧会话检查，不写磁盘。
func (c *acceptanceClient) snapshot(origin string) []*http.Cookie {
	u, _ := url.Parse(origin + "/")
	cookies := c.http.Jar.Cookies(u)
	result := make([]*http.Cookie, 0, len(cookies))
	for _, cookie := range cookies {
		copy := *cookie
		result = append(result, &copy)
	}
	return result
}
