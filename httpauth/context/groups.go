// @Author daixk 2026/06/05
package context

// Auth returns current request auth operations Auth 返回当前请求认证操作入口
func (c *DTokenContext) Auth() *AuthContext {
	return &AuthContext{d: c}
}

// Cookie returns current request cookie operations Cookie 返回当前请求 Cookie 操作入口
func (c *DTokenContext) Cookie() *CookieContext {
	return &CookieContext{d: c}
}

// Access returns current request role and permission operations Access 返回当前请求角色与权限操作入口
func (c *DTokenContext) Access() *AccessContext {
	return &AccessContext{d: c}
}

// Session returns current request session operations Session 返回当前请求会话操作入口
func (c *DTokenContext) Session() *SessionContext {
	return &SessionContext{d: c}
}

// Terminal returns current request terminal operations Terminal 返回当前请求终端操作入口
func (c *DTokenContext) Terminal() *TerminalContext {
	return &TerminalContext{d: c}
}

// Disable returns current request disable operations Disable 返回当前请求封禁操作入口
func (c *DTokenContext) Disable() *DisableContext {
	return &DisableContext{d: c}
}

// Refresh returns refresh token operations Refresh 返回刷新令牌操作入口
func (c *DTokenContext) Refresh() *RefreshContext {
	return &RefreshContext{d: c}
}

// Nonce returns nonce operations Nonce 返回 Nonce 操作入口
func (c *DTokenContext) Nonce() *NonceContext {
	return &NonceContext{d: c}
}

// Ticket returns ticket operations Ticket 返回 Ticket 操作入口
func (c *DTokenContext) Ticket() *TicketContext {
	return &TicketContext{d: c}
}

// ShortKey returns short key operations ShortKey 返回短 Key 操作入口
func (c *DTokenContext) ShortKey() *ShortKeyContext {
	return &ShortKeyContext{d: c}
}

// OAuth2 returns OAuth2 operations OAuth2 返回 OAuth2 操作入口
func (c *DTokenContext) OAuth2() *OAuth2Context {
	return &OAuth2Context{d: c}
}

// AuthContext groups current token auth operations AuthContext 聚合当前 Token 认证操作
type AuthContext struct {
	d *DTokenContext
}

// CookieContext groups current request cookie operations CookieContext 聚合当前请求 Cookie 操作
type CookieContext struct {
	d *DTokenContext
}

// AccessContext groups current role and permission operations AccessContext 聚合当前角色与权限操作
type AccessContext struct {
	d *DTokenContext
}

// SessionContext groups current session operations SessionContext 聚合当前会话操作
type SessionContext struct {
	d *DTokenContext
}

// TerminalContext groups current terminal operations TerminalContext 聚合当前终端操作
type TerminalContext struct {
	d *DTokenContext
}

// DisableContext groups current disable operations DisableContext 聚合当前封禁操作
type DisableContext struct {
	d *DTokenContext
}

// RefreshContext groups refresh token operations RefreshContext 聚合刷新令牌操作
type RefreshContext struct {
	d *DTokenContext
}

// NonceContext groups nonce operations NonceContext 聚合 Nonce 操作
type NonceContext struct {
	d *DTokenContext
}

// TicketContext groups ticket operations TicketContext 聚合 Ticket 操作
type TicketContext struct {
	d *DTokenContext
}

// ShortKeyContext groups short key operations ShortKeyContext 聚合短 Key 操作
type ShortKeyContext struct {
	d *DTokenContext
}

// OAuth2Context groups OAuth2 operations OAuth2Context 聚合 OAuth2 操作
type OAuth2Context struct {
	d *DTokenContext
}
