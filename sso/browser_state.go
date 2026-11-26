package sso

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Zany2/dtoken-go/contract"
)

// browserLoginState retains the return path separately from untrusted callback parameters. browserLoginState 独立保存返回路径，避免依赖不可信回调参数。
type browserLoginState struct {
	// CodeVerifier is private server-side proof, consumed with the browser-bound flow. CodeVerifier 是服务端私有证明，与浏览器绑定流程一起消费。
	CodeVerifier string `json:"codeVerifier,omitempty"`
	// Binding stores the browser secret hash. Binding 保存浏览器秘密值的哈希。
	Binding string `json:"binding"`
	// ReturnTo retains the validated application destination. ReturnTo 保存已校验的应用返回地址。
	ReturnTo string `json:"returnTo"`
	// ExpiresAt enforces a deadline independently of backend cleanup. ExpiresAt 独立于后端清理执行截止时间限制。
	ExpiresAt time.Time `json:"expiresAt"`
}

// browserBindingHash avoids storing browser cookie secrets in the state record. browserBindingHash 避免在状态记录中保存浏览器 Cookie 秘密值。
func browserBindingHash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

// validBrowserStateID bounds cookie names and storage keys to generated random identifiers. validBrowserStateID 将 Cookie 名及存储键限制为生成的随机标识。
func validBrowserStateID(value string) bool {
	if len(value) < 26 || len(value) > 128 {
		return false
	}
	for _, ch := range value {
		if !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')) {
			return false
		}
	}
	return true
}

// stateCookie isolates parallel login flows so one callback cannot clear another tab's state. stateCookie 隔离并行登录流程，避免一次回调清除其他标签页状态。
func (b *BrowserClient) stateCookie(state string) CookieOptions {
	options := b.options.StateCookie
	options.Name += "_" + state
	return options
}

// decodeState validates records independently of storage TTL. decodeState 独立于存储 TTL 校验状态记录。
func (b *BrowserClient) decodeState(raw any, binding string) (*browserLoginState, error) {
	var payload []byte
	switch value := raw.(type) {
	case string:
		payload = []byte(value)
	case []byte:
		payload = value
	default:
		return nil, ErrInvalidLoginState
	}
	var record browserLoginState
	if err := json.Unmarshal(payload, &record); err != nil || record.Binding != browserBindingHash(binding) || !time.Now().Before(record.ExpiresAt) || !validBrowserReturnTo(record.ReturnTo) {
		return nil, ErrInvalidLoginState
	}
	if (b.options.PKCE && !validPKCEVerifier(record.CodeVerifier)) || (!b.options.PKCE && record.CodeVerifier != "") {
		return nil, ErrInvalidLoginState
	}
	return &record, nil
}

// consumeState verifies browser binding before atomic consumption, with an ordinary-storage fallback. consumeState 在原子消费前验证浏览器绑定，并提供普通存储回退。
func (b *BrowserClient) consumeState(r *http.Request, state string) (*browserLoginState, error) {
	cookie := b.stateCookie(state)
	if len(r.CookiesNamed(cookie.Name)) != 1 {
		return nil, ErrInvalidLoginState
	}
	binding, ok := LoginIDFromCookie(cookie)(r)
	if !ok {
		return nil, ErrInvalidLoginState
	}
	atomicStorage, atomic := b.options.Storage.(contract.AtomicStorage)
	if !atomic {
		b.mu.Lock()
		defer b.mu.Unlock()
	}
	key := b.options.KeyPrefix + state
	raw, err := b.options.Storage.Get(r.Context(), key)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStorageUnavailable, err)
	}
	record, err := b.decodeState(raw, binding)
	if err != nil {
		return nil, err
	}
	if atomic {
		raw, err = atomicStorage.GetAndDelete(r.Context(), key)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrStorageUnavailable, err)
		}
		return b.decodeState(raw, binding)
	}
	if err := b.options.Storage.Delete(r.Context(), key); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStorageUnavailable, err)
	}
	return record, nil
}
