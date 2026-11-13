package config

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

// TestValidateTokenNameBySource verifies HTTP name restrictions without limiting query/body names. TestValidateTokenNameBySource 验证 HTTP 名称限制，同时保留 Query/Body 名称的兼容性。
func TestValidateTokenNameBySource(t *testing.T) {
	sources := []struct {
		name                        string
		header, cookie, query, body bool
		requireHTTP                 bool
	}{
		{name: "header", header: true, requireHTTP: true},
		{name: "cookie", cookie: true, requireHTTP: true},
		{name: "query", query: true},
		{name: "body", body: true},
		{name: "header and query", header: true, query: true, requireHTTP: true},
		{name: "cookie and body", cookie: true, body: true, requireHTTP: true},
	}
	names := []struct {
		name, value string
		invalidHTTP bool
	}{
		{name: "default", value: "dtoken"},
		{name: "authorization", value: "Authorization"},
		{name: "hyphen and underscore", value: "X-Auth_Token"},
		{name: "allowed punctuation", value: "!#$%&'*+-.^_`|~"},
		{name: "colon", value: "access:token", invalidHTTP: true},
		{name: "equals", value: "a=b", invalidHTTP: true},
		{name: "semicolon", value: "a;b", invalidHTTP: true},
		{name: "slash", value: "a/b", invalidHTTP: true},
		{name: "unicode", value: "令牌", invalidHTTP: true},
		{name: "null byte", value: "a\x00b", invalidHTTP: true},
		{name: "delete byte", value: "a\x7fb", invalidHTTP: true},
	}

	for _, source := range sources {
		for _, name := range names {
			t.Run(source.name+"/"+name.name, func(t *testing.T) {
				cfg := DefaultConfig()
				cfg.IsReadHeader, cfg.IsReadCookie = source.header, source.cookie
				cfg.IsReadQuery, cfg.IsReadBody = source.query, source.body
				cfg.TokenName = name.value

				err := cfg.Validate()
				if source.requireHTTP && name.invalidHTTP {
					if err == nil || !strings.Contains(err.Error(), "Config.TokenName") {
						t.Fatalf("Validate() error = %v, want TokenName error", err)
					}
				} else if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
			})
		}
	}
}

// TestValidateCookieMaxAgeBounds verifies conversion to int on both 32-bit and 64-bit platforms. TestValidateCookieMaxAgeBounds 验证 32 位和 64 位平台转换为 int 时的范围。
func TestValidateCookieMaxAgeBounds(t *testing.T) {
	tests := []struct {
		name    string
		value   int64
		wantErr bool
	}{
		{name: "negative", value: -1, wantErr: true},
		{name: "session", value: 0},
		{name: "positive", value: 3600},
		{name: "int32 max", value: math.MaxInt32},
		{name: "above int32 max", value: int64(math.MaxInt32) + 1, wantErr: strconv.IntSize == 32},
		{name: "int64 max", value: math.MaxInt64, wantErr: strconv.IntSize == 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.IsReadCookie = true
			cfg.CookieConfig.MaxAge = tt.value

			err := cfg.Validate()
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "CookieConfig.MaxAge") {
					t.Fatalf("Validate() error = %v, want MaxAge error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if got := int64(int(cfg.CookieConfig.MaxAge)); got != tt.value {
				t.Fatalf("MaxAge after conversion = %d, want %d", got, tt.value)
			}
		})
	}
}
