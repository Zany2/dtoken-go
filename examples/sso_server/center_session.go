package main

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"net/http"

	dtoken "github.com/Zany2/dtoken-go"
	"github.com/Zany2/dtoken-go/sso"
)

// demoCenter composes application-owned login state with the SSO protocol. demoCenter 将应用管理的登录态与 SSO 协议组合。
type demoCenter struct {
	auth *dtoken.Auth // auth owns the center's server-side login state. auth 管理中心的服务端登录态。
}

// newCenterAuth builds independent, fixed-lifetime browser sessions. newCenterAuth 构建独立且有效期固定的浏览器会话。
func newCenterAuth() (*dtoken.Auth, error) {
	return dtoken.NewBuilder().AuthType("sso-center").TimeoutDuration(cookie.MaxAge).
		AutoRenew(false).IsConcurrent(true).IsShare(false).IsLog(false).IsPrintBanner(false).BuildInstance()
}

// loginID resolves the signed Cookie through the same Auth used for login and logout. loginID 通过登录与注销共用的 Auth 解析签名 Cookie。
func (c *demoCenter) loginID(r *http.Request) (string, bool) {
	token, ok := sso.LoginIDFromCookie(cookie)(r)
	if !ok {
		return "", false
	}
	loginID, err := c.auth.GetLoginID(r.Context(), token)
	return loginID, err == nil && loginID != ""
}

// startSession replaces this browser's prior center login after creating the new session. startSession 创建新会话后替换当前浏览器原有的中心登录。
func (c *demoCenter) startSession(w http.ResponseWriter, r *http.Request, loginID string) error {
	token, err := c.auth.Login(r.Context(), dtoken.LoginOptions{LoginID: loginID, Device: "browser", DeviceID: rand.Text()})
	if err != nil {
		return err
	}
	if previous, ok := sso.LoginIDFromCookie(cookie)(r); ok {
		if err := c.auth.LogoutByToken(r.Context(), previous); err != nil {
			return err
		}
	}
	sso.SetLoginIDCookie(w, cookie, token)
	return nil
}

// centerSessionRef exposes a non-bearer reference instead of the center login token. centerSessionRef 暴露不可直接用于登录的引用，不向子应用泄露中心 Token。
func centerSessionRef(token string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(token)))
}

// authorizationExtra binds each Ticket to the exact authenticated center session. authorizationExtra 将每张 Ticket 绑定到签发时的具体中心会话。
func (c *demoCenter) authorizationExtra(r *http.Request, _, loginID string) (map[string]any, error) {
	token, ok := sso.LoginIDFromCookie(cookie)(r)
	if !ok {
		return nil, sso.ErrInvalidTicket
	}
	actual, err := c.auth.GetLoginID(r.Context(), token)
	if err != nil {
		return nil, err
	}
	if actual != loginID {
		return nil, sso.ErrInvalidTicket
	}
	return map[string]any{"centerSessionID": centerSessionRef(token)}, nil
}

// centerSession resolves the original session without renewing it or hiding storage failures. centerSession 解析原中心会话，不续期且不掩盖存储故障。
func (c *demoCenter) centerSession(r *http.Request, loginID string, extra map[string]any) (*dtoken.TokenIntrospection, error) {
	ref, _ := extra["centerSessionID"].(string)
	if ref == "" {
		return nil, nil
	}

	// Use the account index to find the exact token, then validate only that session. 通过账号索引找到具体 Token，再校验该会话。
	tokens, err := c.auth.GetTokenValueListByLoginID(r.Context(), loginID, false)
	if err != nil {
		return nil, err
	}
	for _, token := range tokens {
		if centerSessionRef(token) != ref {
			continue
		}
		info, err := c.auth.IntrospectToken(r.Context(), token)
		if err != nil {
			return nil, err
		}

		// The demo has a fixed lifetime; zero remaining seconds cannot be sent as an unspecified TTL. 示例使用固定有效期，剩余零秒不能作为未指定 TTL 返回。
		if !info.Active || info.LoginID != loginID || info.ExpiresIn <= 0 {
			return nil, nil
		}
		return info, nil
	}
	return nil, nil
}

// enrichExchange rejects a retired center session and supplies its remaining token lifetime. enrichExchange 拒绝已失效的中心会话，并提供其 Token 剩余有效期。
func (c *demoCenter) enrichExchange(r *http.Request, _ string, result *sso.TicketExchangeResult) error {
	info, err := c.centerSession(r, result.LoginID, result.Extra)
	if err != nil {
		return err
	}
	if info == nil {
		return sso.ErrInvalidTicket
	}
	result.RemainTokenTimeout = info.ExpiresIn
	result.DeviceID = info.DeviceID
	return nil
}

// checkCredentialActive applies the same center-session rule to introspection and userinfo. checkCredentialActive 为 introspect 和 userinfo 应用相同的中心会话规则。
func (c *demoCenter) checkCredentialActive(r *http.Request, credential sso.CredentialInfo) (bool, error) {
	info, err := c.centerSession(r, credential.LoginID, credential.Extra)
	return info != nil, err
}

// logout revokes all center browser sessions for the resolved account. logout 撤销已解析账号的所有中心浏览器会话。
func (c *demoCenter) logout(r *http.Request, loginID string) error {
	return c.auth.LogoutByLoginID(r.Context(), loginID)
}
