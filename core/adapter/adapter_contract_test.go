package adapter

import (
	"reflect"
	"testing"
	"time"
)

// TestTTLSentinelValues verifies the shared TTL sentinel values. TestTTLSentinelValues 验证统一 TTL 哨兵值。
func TestTTLSentinelValues(t *testing.T) {
	if TTLNoExpire != time.Duration(-1) {
		t.Fatalf("TTLNoExpire = %v, want -1ns", TTLNoExpire)
	}
	if TTLNotFound != time.Duration(-2) {
		t.Fatalf("TTLNotFound = %v, want -2ns", TTLNotFound)
	}
	if TTLNoExpire == TTLNotFound {
		t.Fatal("TTL sentinel values must be distinct")
	}
	if TTLNoExpire >= 0 || TTLNotFound >= 0 {
		t.Fatal("TTL sentinel values must be negative")
	}
}

// TestTokenStyleValues verifies every built-in token style is non-empty and unique. TestTokenStyleValues 验证所有内置 Token 风格非空且唯一。
func TestTokenStyleValues(t *testing.T) {
	styles := []struct {
		style TokenStyle
		want  string
	}{
		{style: TokenStyleUUID, want: "uuid"},
		{style: TokenStyleSimple, want: "simple"},
		{style: TokenStyleRandom32, want: "random32"},
		{style: TokenStyleRandom64, want: "random64"},
		{style: TokenStyleRandom128, want: "random128"},
		{style: TokenStyleJWT, want: "jwt"},
		{style: TokenStyleHash, want: "hash"},
		{style: TokenStyleTimestamp, want: "timestamp"},
		{style: TokenStyleTik, want: "tik"},
	}

	seen := make(map[TokenStyle]struct{}, len(styles))
	for _, tt := range styles {
		if tt.style == "" {
			t.Fatal("token style must not be empty")
		}
		if string(tt.style) != tt.want {
			t.Fatalf("token style = %q, want %q", tt.style, tt.want)
		}
		if _, exists := seen[tt.style]; exists {
			t.Fatalf("duplicate token style %q", tt.style)
		}
		seen[tt.style] = struct{}{}
	}
}

// TestAdapterInterfaceMethodSets verifies the public interface contracts stay complete. TestAdapterInterfaceMethodSets 验证公开接口方法集合保持完整。
func TestAdapterInterfaceMethodSets(t *testing.T) {
	tests := []struct {
		name    string
		iface   reflect.Type
		methods []string
	}{
		{name: "Codec", iface: reflect.TypeOf((*Codec)(nil)).Elem(), methods: []string{"Name", "Encode", "Decode"}},
		{name: "Generator", iface: reflect.TypeOf((*Generator)(nil)).Elem(), methods: []string{"Generate"}},
		{name: "Pool", iface: reflect.TypeOf((*Pool)(nil)).Elem(), methods: []string{"Submit", "Stop", "Stats"}},
		{name: "Storage", iface: reflect.TypeOf((*Storage)(nil)).Elem(), methods: []string{"Set", "Get", "Delete", "Exists", "Expire", "TTL", "Ping"}},
		{name: "AtomicStorage", iface: reflect.TypeOf((*AtomicStorage)(nil)).Elem(), methods: []string{"GetAndDelete", "SetIfAbsent"}},
		{name: "ScannerStorage", iface: reflect.TypeOf((*ScannerStorage)(nil)).Elem(), methods: []string{"Keys"}},
		{name: "AdminStorage", iface: reflect.TypeOf((*AdminStorage)(nil)).Elem(), methods: []string{"Clear"}},
		{name: "FullStorage", iface: reflect.TypeOf((*FullStorage)(nil)).Elem(), methods: []string{"Set", "Get", "Delete", "Exists", "Expire", "TTL", "Ping", "GetAndDelete", "SetIfAbsent", "Keys", "Clear"}},
		{name: "Log", iface: reflect.TypeOf((*Log)(nil)).Elem(), methods: []string{"Print", "Printf", "Debug", "Debugf", "Info", "Infof", "Warn", "Warnf", "Error", "Errorf"}},
		{name: "LogControl", iface: reflect.TypeOf((*LogControl)(nil)).Elem(), methods: []string{"Print", "Printf", "Debug", "Debugf", "Info", "Infof", "Warn", "Warnf", "Error", "Errorf", "Close", "Flush", "SetLevel", "SetPrefix", "SetStdout", "LogPath", "DropCount"}},
		{name: "RequestContext", iface: reflect.TypeOf((*RequestContext)(nil)).Elem(), methods: []string{"GetHeader", "GetHeaders", "GetQuery", "GetQueryAll", "GetPostForm", "GetCookie", "GetBody", "GetClientIP", "GetMethod", "GetPath", "GetURL", "GetUserAgent", "IsTLS", "SetStatusCode", "SetHeader", "Write", "SetCookie", "SetCookieWithOptions", "Set", "Get", "GetString", "MustGet", "Abort", "IsAborted"}},
		{name: "RequestContextExt", iface: reflect.TypeOf((*RequestContextExt)(nil)).Elem(), methods: []string{"GetHeader", "GetHeaders", "GetQuery", "GetQueryAll", "GetPostForm", "GetCookie", "GetBody", "GetClientIP", "GetMethod", "GetPath", "GetURL", "GetUserAgent", "IsTLS", "SetStatusCode", "SetHeader", "Write", "SetCookie", "SetCookieWithOptions", "Set", "Get", "GetString", "MustGet", "Abort", "IsAborted", "JSON", "GetRawRequest", "GetRawResponseWriter"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.iface.NumMethod() != len(tt.methods) {
				t.Fatalf("NumMethod() = %d, want %d", tt.iface.NumMethod(), len(tt.methods))
			}
			for _, method := range tt.methods {
				if _, ok := tt.iface.MethodByName(method); !ok {
					t.Fatalf("missing method %q", method)
				}
			}
		})
	}
}
