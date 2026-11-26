package sso

import (
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

// normalizedHTTPOrigin compares HTTP origins using host case and effective ports without changing paths or redirects. normalizedHTTPOrigin 按主机大小写及有效端口比较 HTTP 来源，不改变路径或重定向地址。
func normalizedHTTPOrigin(u *url.URL) string {
	if u == nil || u.User != nil || u.Opaque != "" || u.Hostname() == "" || hasConfigWhitespace(u.Host) || strings.ContainsAny(u.Host, "\\%") || strings.HasSuffix(u.Host, ":") {
		return ""
	}
	scheme := strings.ToLower(u.Scheme)
	port := 0
	switch scheme {
	case "http":
		port = 80
	case "https":
		port = 443
	default:
		return ""
	}
	if u.Port() != "" {
		var err error
		port, err = strconv.Atoi(u.Port())
		if err != nil || port < 1 || port > 65535 {
			return ""
		}
	}
	host := strings.ToLower(u.Hostname())
	if address, err := netip.ParseAddr(host); err == nil {
		host = address.String()
	}
	return scheme + "://" + net.JoinHostPort(host, strconv.Itoa(port))
}

// browserRequestOrigin accepts one serialized Origin value, never a URL with a path or credentials. browserRequestOrigin 仅接受序列化的单个 Origin，不接受携带路径或凭据的 URL。
func browserRequestOrigin(value string) string {
	u, err := url.Parse(value)
	if err != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || strings.Contains(value, "#") {
		return ""
	}
	return normalizedHTTPOrigin(u)
}
