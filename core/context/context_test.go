package context

import (
	"testing"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/manager"
)

// TestGetTokenValuePrecedence verifies header, bearer, cookie, and body lookup order. TestGetTokenValuePrecedence 验证 Header、Bearer、Cookie 和 Body 的读取顺序。
func TestGetTokenValuePrecedence(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.TokenName = "X-Token"
	cfg.IsReadHeader = true
	cfg.IsReadCookie = true
	cfg.IsReadBody = true
	mgr := manager.NewManager(cfg, nil, nil, nil, nil, nil, nil)

	req := &testRequestContext{
		headers: map[string]string{
			"X-Token":       " header-token ",
			"Authorization": "Bearer bearer-token",
		},
		cookies: map[string]string{"X-Token": "cookie-token"},
		forms:   map[string]string{"X-Token": "body-token"},
	}
	if token := NewContext(req, mgr).GetTokenValue(); token != "header-token" {
		t.Fatalf("GetTokenValue() = %q, want header-token", token)
	}

	delete(req.headers, "X-Token")
	if token := NewContext(req, mgr).GetTokenValue(); token != "bearer-token" {
		t.Fatalf("GetTokenValue() = %q, want bearer-token", token)
	}

	delete(req.headers, "Authorization")
	if token := NewContext(req, mgr).GetTokenValue(); token != "cookie-token" {
		t.Fatalf("GetTokenValue() = %q, want cookie-token", token)
	}

	delete(req.cookies, "X-Token")
	if token := NewContext(req, mgr).GetTokenValue(); token != "body-token" {
		t.Fatalf("GetTokenValue() = %q, want body-token", token)
	}
}

// TestGetTokenValueParsesConfiguredAuthorizationHeader verifies TokenName=Authorization still parses bearer. TestGetTokenValueParsesConfiguredAuthorizationHeader 验证 TokenName=Authorization 时仍解析 Bearer。
func TestGetTokenValueParsesConfiguredAuthorizationHeader(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.TokenName = "Authorization"
	cfg.IsReadHeader = true
	mgr := manager.NewManager(cfg, nil, nil, nil, nil, nil, nil)

	req := &testRequestContext{
		headers: map[string]string{
			"Authorization": "Bearer auth-token",
		},
	}

	if token := NewContext(req, mgr).GetTokenValue(); token != "auth-token" {
		t.Fatalf("GetTokenValue() = %q, want auth-token", token)
	}
}

// TestExtractBearerToken verifies bearer parsing compatibility. TestExtractBearerToken 验证 Bearer 解析兼容性。
func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name string
		auth string
		want string
	}{
		{name: "bearer", auth: "Bearer abc", want: "abc"},
		{name: "case insensitive", auth: "bearer abc", want: "abc"},
		{name: "raw compatibility", auth: "raw-token", want: "raw-token"},
		{name: "empty", auth: "  ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractBearerToken(tt.auth); got != tt.want {
				t.Fatalf("extractBearerToken(%q) = %q, want %q", tt.auth, got, tt.want)
			}
		})
	}
}

type testRequestContext struct {
	headers map[string]string
	cookies map[string]string
	forms   map[string]string
	values  map[string]any
}

func (c *testRequestContext) GetHeader(key string) string { return c.headers[key] }
func (c *testRequestContext) GetHeaders() map[string][]string {
	result := map[string][]string{}
	for key, value := range c.headers {
		result[key] = []string{value}
	}
	return result
}
func (c *testRequestContext) GetQuery(string) string           { return "" }
func (c *testRequestContext) GetQueryAll() map[string][]string { return nil }
func (c *testRequestContext) GetPostForm(key string) string    { return c.forms[key] }
func (c *testRequestContext) GetCookie(key string) string      { return c.cookies[key] }
func (c *testRequestContext) GetBody() ([]byte, error)         { return nil, nil }
func (c *testRequestContext) GetClientIP() string              { return "" }
func (c *testRequestContext) GetMethod() string                { return "" }
func (c *testRequestContext) GetPath() string                  { return "" }
func (c *testRequestContext) GetURL() string                   { return "" }
func (c *testRequestContext) GetUserAgent() string             { return "" }
func (c *testRequestContext) IsTLS() bool                      { return false }
func (c *testRequestContext) SetStatusCode(int)                {}
func (c *testRequestContext) SetHeader(string, string)         {}
func (c *testRequestContext) Write(data []byte) (int, error)   { return len(data), nil }
func (c *testRequestContext) SetCookie(string, string, int, string, string, bool, bool) {
}
func (c *testRequestContext) SetCookieWithOptions(*adapter.CookieOptions) {}
func (c *testRequestContext) Set(key string, value any) {
	if c.values == nil {
		c.values = map[string]any{}
	}
	c.values[key] = value
}
func (c *testRequestContext) Get(key string) (any, bool) {
	value, ok := c.values[key]
	return value, ok
}
func (c *testRequestContext) GetString(key string) string {
	value, _ := c.values[key].(string)
	return value
}
func (c *testRequestContext) MustGet(key string) any { return c.values[key] }
func (c *testRequestContext) Abort()                 {}
func (c *testRequestContext) IsAborted() bool        { return false }
