// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/sso"
)

// RegisterSSOClient registers an SSO client. RegisterSSOClient 注册 SSO 客户端。
func (m *Manager) RegisterSSOClient(client *sso.Client) error {
	return m.ssoManager.RegisterClient(client)
}

// UnregisterSSOClient unregisters an SSO client. UnregisterSSOClient 注销 SSO 客户端。
func (m *Manager) UnregisterSSOClient(clientID string) error {
	return m.ssoManager.UnregisterClient(clientID)
}

// GetSSOClient gets an SSO client by id. GetSSOClient 根据 ID 获取 SSO 客户端。
func (m *Manager) GetSSOClient(clientID string) (*sso.Client, error) {
	return m.ssoManager.GetClient(clientID)
}

// GenerateSSOTicket generates a one-time SSO ticket. GenerateSSOTicket 生成一次性 SSO Ticket。
func (m *Manager) GenerateSSOTicket(ctx context.Context, clientID, loginID, redirectURI string, scopes []string, extra map[string]any) (*sso.Ticket, error) {
	return m.ssoManager.GenerateTicket(ctx, clientID, loginID, redirectURI, scopes, extra)
}

// GenerateSSOTicketWithTimeout generates a one-time SSO ticket with timeout. GenerateSSOTicketWithTimeout 使用指定有效期生成一次性 SSO Ticket。
func (m *Manager) GenerateSSOTicketWithTimeout(ctx context.Context, clientID, loginID, redirectURI string, scopes []string, extra map[string]any, timeout time.Duration) (*sso.Ticket, error) {
	return m.ssoManager.GenerateTicketWithTimeout(ctx, clientID, loginID, redirectURI, scopes, extra, timeout)
}

// ValidateSSOTicket validates an SSO ticket without consuming it. ValidateSSOTicket 校验 SSO Ticket 但不消费。
func (m *Manager) ValidateSSOTicket(ctx context.Context, ticket string) (*sso.Ticket, error) {
	return m.ssoManager.ValidateTicket(ctx, ticket)
}

// ConsumeSSOTicket validates and consumes an SSO ticket. ConsumeSSOTicket 校验并消费 SSO Ticket。
func (m *Manager) ConsumeSSOTicket(ctx context.Context, ticket, clientID, clientSecret, redirectURI string) (*sso.Ticket, error) {
	return m.ssoManager.ConsumeTicket(ctx, ticket, clientID, clientSecret, redirectURI)
}

// RevokeSSOTicket revokes an SSO ticket. RevokeSSOTicket 撤销 SSO Ticket。
func (m *Manager) RevokeSSOTicket(ctx context.Context, ticket string) error {
	return m.ssoManager.RevokeTicket(ctx, ticket)
}

// GetSSOTicketTTL returns SSO ticket TTL in seconds. GetSSOTicketTTL 获取 SSO Ticket 剩余秒数。
func (m *Manager) GetSSOTicketTTL(ctx context.Context, ticket string) (int64, error) {
	return m.ssoManager.GetTicketTTL(ctx, ticket)
}
