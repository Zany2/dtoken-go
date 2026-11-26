package acceptance

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// acceptanceTransport provides deterministic responses without network access. acceptanceTransport 无需网络即可提供确定响应。
type acceptanceTransport func(*http.Request) (*http.Response, error)

// RoundTrip calls the injected transport. RoundTrip 调用注入的传输函数。
func (f acceptanceTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestAcceptanceDestinations rejects unsafe destinations before sending credentials or cookies. TestAcceptanceDestinations 在发送凭据或 Cookie 前拒绝不安全目标。
func TestAcceptanceDestinations(t *testing.T) {
	for _, value := range []string{"", "http://remote.example", "https://user:secret@app.example", "https://app.example/path", "https://app.example?", "https://app.example#", "ftp://localhost"} {
		if _, err := acceptanceOrigin(value); err == nil {
			t.Fatal("invalid origin accepted")
		}
	}
	for _, value := range []string{"http://localhost:9200", "http://127.0.0.1:9200", "http://[::1]:9200", "https://app.example"} {
		if _, err := acceptanceOrigin(value); err != nil {
			t.Fatalf("valid origin rejected: %v", err)
		}
	}
	c, err := newAcceptanceClient("https://center.example", "https://app.example")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	c.http.Transport = acceptanceTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("unexpected transport")
	})
	for _, target := range []string{"https://outside.example/login", "http://center.example/login", "https://user:secret@center.example/login", "https://center.example/login#fragment"} {
		if _, err := c.request(http.MethodPost, target, url.Values{"password": {"private-value"}}, "", nil); err == nil {
			t.Fatal("unsafe target accepted")
		}
	}
	if calls != 0 {
		t.Fatal("unsafe target reached transport")
	}
	for _, location := range []string{"//outside.example/sso/callback", "http://app.example/sso/callback", "https://app.example/other", "https://user:secret@app.example/sso/callback", "https://app.example/sso/callback#fragment"} {
		response := &acceptanceResponse{status: http.StatusFound, header: http.Header{"Location": {location}}}
		if _, err := c.redirect("https://center.example/sso/authorize", response, "https://app.example", "/sso/callback"); err == nil {
			t.Fatal("unsafe redirect accepted")
		}
	}
}

// TestAcceptanceTransportBoundaries verifies redirect, body and diagnostic limits with an in-memory transport. TestAcceptanceTransportBoundaries 使用内存传输验证重定向、响应体及诊断限制。
func TestAcceptanceTransportBoundaries(t *testing.T) {
	c, err := newAcceptanceClient("https://center.example")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	c.http.Transport = acceptanceTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {"https://outside.example/private"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	response, err := c.request(http.MethodGet, "https://center.example/start", nil, "", nil)
	if err != nil || response.status != http.StatusFound || calls != 1 {
		t.Fatal("redirect was followed implicitly")
	}
	c.http.Transport = acceptanceTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("private-value") })
	if _, err := c.request(http.MethodGet, "https://center.example/?secret=private-value", nil, "", nil); err == nil || strings.Contains(err.Error(), "private-value") {
		t.Fatal("transport diagnostic leaked")
	}
	c.http.Transport = acceptanceTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(strings.Repeat("a", (1<<20)+1)))}, nil
	})
	if _, err := c.request(http.MethodGet, "https://center.example/", nil, "", nil); err == nil {
		t.Fatal("unbounded response accepted")
	}
}
