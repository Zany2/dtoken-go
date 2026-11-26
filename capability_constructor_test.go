package dtoken_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	djson "github.com/Zany2/dtoken-go/codec/json"
	adapter "github.com/Zany2/dtoken-go/contract"
	derror "github.com/Zany2/dtoken-go/errors"
	"github.com/Zany2/dtoken-go/nonce"
	"github.com/Zany2/dtoken-go/oauth2"
	"github.com/Zany2/dtoken-go/shortkey"
	"github.com/Zany2/dtoken-go/storage/memory"
	"github.com/Zany2/dtoken-go/ticket"
)

// TestCapabilityConstructorsRejectMissingComponents covers direct, default, and configured public constructors. TestCapabilityConstructorsRejectMissingComponents 覆盖直接、默认及配置构造器的空组件检查。
func TestCapabilityConstructorsRejectMissingComponents(t *testing.T) {
	storage := memory.NewStorage()
	codec := djson.NewJSONSerializer()
	var nilStorage *memory.Storage
	var nilCodec *djson.JSONSerializer
	for _, constructor := range []struct {
		name       string
		needsCodec bool
		build      func(adapter.Storage, adapter.Codec) (any, error)
	}{
		{"nonce/default", false, func(s adapter.Storage, c adapter.Codec) (any, error) {
			return nonce.NewDefaultNonceManager("user", "dt", s)
		}},
		{"nonce/configured", false, func(s adapter.Storage, c adapter.Codec) (any, error) {
			return nonce.NewNonceManagerWithConfig("user", "dt", s, nil)
		}},
		{"nonce/direct", false, func(s adapter.Storage, c adapter.Codec) (any, error) {
			return nonce.NewNonceManager("user", "dt", s, time.Minute)
		}},
		{"oauth2/default", true, func(s adapter.Storage, c adapter.Codec) (any, error) {
			return oauth2.NewDefaultOAuth2Server("user", "dt", s, c)
		}},
		{"oauth2/configured", true, func(s adapter.Storage, c adapter.Codec) (any, error) {
			return oauth2.NewOAuth2ServerWithConfig("user", "dt", s, c, nil)
		}},
		{"oauth2/direct", true, func(s adapter.Storage, c adapter.Codec) (any, error) {
			return oauth2.NewOAuth2Server("user", "dt", s, c)
		}},
		{"ticket/default", true, func(s adapter.Storage, c adapter.Codec) (any, error) {
			return ticket.NewDefaultManager("user", "dt", s, c)
		}},
		{"ticket/configured", true, func(s adapter.Storage, c adapter.Codec) (any, error) {
			return ticket.NewManagerWithConfig("user", "dt", s, c, nil)
		}},
		{"shortkey/default", true, func(s adapter.Storage, c adapter.Codec) (any, error) {
			return shortkey.NewDefaultManager("user", "dt", s, c)
		}},
		{"shortkey/configured", true, func(s adapter.Storage, c adapter.Codec) (any, error) {
			return shortkey.NewManagerWithConfig("user", "dt", s, c, nil)
		}},
	} {
		t.Run(constructor.name, func(t *testing.T) {
			for _, tc := range []struct {
				name    string
				storage adapter.Storage
				codec   adapter.Codec
				invalid bool
			}{
				{"valid pointers", storage, codec, false},
				{"valid values", struct{ adapter.Storage }{storage}, struct{ adapter.Codec }{codec}, false},
				{"nil storage", nil, codec, true},
				{"typed nil storage", nilStorage, codec, true},
				{"nil codec", storage, nil, constructor.needsCodec},
				{"typed nil codec", storage, nilCodec, constructor.needsCodec},
			} {
				t.Run(tc.name, func(t *testing.T) {
					instance, err := constructor.build(tc.storage, tc.codec)
					present := instance != nil && !reflect.ValueOf(instance).IsNil()
					if tc.invalid {
						if present || !errors.Is(err, derror.ErrInvalidParam) {
							t.Fatalf("constructor = %v, %v, want nil, ErrInvalidParam", instance, err)
						}
					} else if !present || err != nil {
						t.Fatalf("constructor = %v, %v, want usable instance", instance, err)
					}
				})
			}
		})
	}
}

// TestCapabilityConstructorsValidateExplicitConfig prevents silent repair of invalid explicit settings. TestCapabilityConstructorsValidateExplicitConfig 防止显式无效配置被静默替换。
func TestCapabilityConstructorsValidateExplicitConfig(t *testing.T) {
	storage := memory.NewStorage()
	codec := djson.NewJSONSerializer()
	for _, tc := range []struct {
		name  string
		build func() (any, error)
	}{
		{"nonce/zero", func() (any, error) { return nonce.NewNonceManagerWithConfig("user", "dt", storage, &nonce.Config{}) }},
		{"nonce/negative", func() (any, error) { return nonce.NewNonceManager("user", "dt", storage, -time.Second) }},
		{"ticket/zero", func() (any, error) {
			return ticket.NewManagerWithConfig("user", "dt", storage, codec, &ticket.Config{})
		}},
		{"ticket/negative", func() (any, error) {
			return ticket.NewManagerWithConfig("user", "dt", storage, codec, &ticket.Config{TTL: -time.Second})
		}},
		{"shortkey/ttl", func() (any, error) {
			return shortkey.NewManagerWithConfig("user", "dt", storage, codec, &shortkey.Config{Length: 8, MaxGenerateRetries: 8})
		}},
		{"shortkey/length", func() (any, error) {
			return shortkey.NewManagerWithConfig("user", "dt", storage, codec, &shortkey.Config{TTL: time.Minute, MaxGenerateRetries: 8})
		}},
		{"shortkey/retries", func() (any, error) {
			return shortkey.NewManagerWithConfig("user", "dt", storage, codec, &shortkey.Config{TTL: time.Minute, Length: 8})
		}},
		{"oauth2/zero", func() (any, error) {
			return oauth2.NewOAuth2ServerWithConfig("user", "dt", storage, codec, &oauth2.Config{})
		}},
		{"oauth2/refresh_equal", func() (any, error) {
			return oauth2.NewOAuth2ServerWithConfig("user", "dt", storage, codec, &oauth2.Config{CodeExpiration: time.Minute, TokenExpiration: time.Hour, RefreshExpiration: time.Hour})
		}},
		{"oauth2/refresh_shorter", func() (any, error) {
			return oauth2.NewOAuth2ServerWithConfig("user", "dt", storage, codec, &oauth2.Config{CodeExpiration: time.Minute, TokenExpiration: time.Hour, RefreshExpiration: time.Minute})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			instance, err := tc.build()
			if (instance != nil && !reflect.ValueOf(instance).IsNil()) || !errors.Is(err, derror.ErrInvalidParam) {
				t.Fatalf("constructor = %v, %v, want nil, ErrInvalidParam", instance, err)
			}
		})
	}
}
