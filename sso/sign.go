// @Author daixk 2026/05/28
package sso

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
)

// Signer signs and verifies SSO request parameters. Signer 对 SSO 请求参数进行签名与校验。
type Signer struct {
	secret string     // secret stores shared signing secret. secret 存储共享签名密钥。
	params ParamNames // params stores protocol parameter names. params 存储协议参数名。
}

// NewSigner creates a signer with default parameter names. NewSigner 使用默认参数名创建签名器。
func NewSigner(secret string) Signer {
	return Signer{secret: secret, params: DefaultParamNames()}
}

// NewSignerWithParams creates a signer with custom parameter names. NewSignerWithParams 使用自定义参数名创建签名器。
func NewSignerWithParams(secret string, params ParamNames) Signer {
	if params == (ParamNames{}) {
		params = DefaultParamNames()
	}
	return Signer{secret: secret, params: params}
}

// Sign signs params with HMAC-SHA256. Sign 使用 HMAC-SHA256 签名参数。
func (s Signer) Sign(values url.Values) string {
	payload := s.canonical(values)
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// AttachSign adds the signature to params. AttachSign 向参数集合写入签名。
func (s Signer) AttachSign(values url.Values) url.Values {
	copied := cloneValues(values)
	copied.Set(s.params.Sign, s.Sign(copied))
	return copied
}

// Verify checks whether params carry a valid signature. Verify 校验参数签名是否有效。
func (s Signer) Verify(values url.Values) bool {
	got := values.Get(s.params.Sign)
	if got == "" {
		return false
	}
	want := s.Sign(values)
	return hmac.Equal([]byte(got), []byte(want))
}

func (s Signer) canonical(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key == s.params.Sign {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		items := append([]string(nil), values[key]...)
		sort.Strings(items)
		for _, value := range items {
			// URL-encode key and value to prevent signature collision对键值进行URL编码，防止签名碰撞
			parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(value))
		}
	}
	return strings.Join(parts, "&")
}

func cloneValues(values url.Values) url.Values {
	copied := make(url.Values, len(values))
	for key, items := range values {
		copied[key] = append([]string(nil), items...)
	}
	return copied
}
