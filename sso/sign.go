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
	return Signer{secret: secret, params: normalizeParamNames(params)}
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
// Empty secrets and multiple signature values are rejected. 拒绝空密钥及多个签名值。
func (s Signer) Verify(values url.Values) bool {
	if s.secret == "" || len(values[s.params.Sign]) != 1 {
		return false
	}
	got := values.Get(s.params.Sign)
	if got == "" {
		return false
	}
	want := s.Sign(values)
	return hmac.Equal([]byte(got), []byte(want))
}

// canonical builds canonical request parameters for signing. canonical 构建用于签名的规范请求参数。
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
		// Preserve value order because URL.Values.Get reads the first value. 保留同名参数值的顺序，因为 URL.Values.Get 读取第一个值。
		for _, value := range values[key] {
			// URL-encode key and value to prevent signature collision对键值进行URL编码，防止签名碰撞
			parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(value))
		}
	}
	return strings.Join(parts, "&")
}

// cloneValues clones URL values before mutation. cloneValues 在修改前克隆 URL 参数。
func cloneValues(values url.Values) url.Values {
	copied := make(url.Values, len(values))
	for key, items := range values {
		copied[key] = append([]string(nil), items...)
	}
	return copied
}
