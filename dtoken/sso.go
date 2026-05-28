// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"context"
	"time"

	"github.com/Zany2/dtoken-go/core/sso"
)

// RegisterSSOClient registers an SSO client. RegisterSSOClient 注册 SSO 客户端。
func RegisterSSOClient(client *sso.Client, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RegisterSSOClient(client)
}

// UnregisterSSOClient unregisters an SSO client. UnregisterSSOClient 注销 SSO 客户端。
func UnregisterSSOClient(clientID string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.UnregisterSSOClient(clientID)
}

// GetSSOClient gets an SSO client by id. GetSSOClient 根据 ID 获取 SSO 客户端。
func GetSSOClient(clientID string, authType ...string) (*sso.Client, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GetSSOClient(clientID)
}

// GenerateSSOTicket generates a one-time SSO ticket. GenerateSSOTicket 生成一次性 SSO Ticket。
func GenerateSSOTicket(ctx context.Context, clientID, loginID, redirectURI string, scopes []string, extra map[string]any, authType ...string) (*sso.Ticket, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GenerateSSOTicket(ctx, clientID, loginID, redirectURI, scopes, extra)
}

// GenerateSSOTicketWithTimeout generates a one-time SSO ticket with timeout. GenerateSSOTicketWithTimeout 使用指定有效期生成一次性 SSO Ticket。
func GenerateSSOTicketWithTimeout(ctx context.Context, clientID, loginID, redirectURI string, scopes []string, extra map[string]any, timeout time.Duration, authType ...string) (*sso.Ticket, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.GenerateSSOTicketWithTimeout(ctx, clientID, loginID, redirectURI, scopes, extra, timeout)
}

// ValidateSSOTicket validates an SSO ticket without consuming it. ValidateSSOTicket 校验 SSO Ticket 但不消费。
func ValidateSSOTicket(ctx context.Context, ticket string, authType ...string) (*sso.Ticket, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ValidateSSOTicket(ctx, ticket)
}

// ConsumeSSOTicket validates and consumes an SSO ticket. ConsumeSSOTicket 校验并消费 SSO Ticket。
func ConsumeSSOTicket(ctx context.Context, ticket, clientID, clientSecret, redirectURI string, authType ...string) (*sso.Ticket, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return nil, err
	}
	return mgr.ConsumeSSOTicket(ctx, ticket, clientID, clientSecret, redirectURI)
}

// RevokeSSOTicket revokes an SSO ticket. RevokeSSOTicket 撤销 SSO Ticket。
func RevokeSSOTicket(ctx context.Context, ticket string, authType ...string) error {
	mgr, err := GetManager(authType...)
	if err != nil {
		return err
	}
	return mgr.RevokeSSOTicket(ctx, ticket)
}

// GetSSOTicketTTL returns SSO ticket TTL in seconds. GetSSOTicketTTL 获取 SSO Ticket 剩余秒数。
func GetSSOTicketTTL(ctx context.Context, ticket string, authType ...string) (int64, error) {
	mgr, err := GetManager(authType...)
	if err != nil {
		return 0, err
	}
	return mgr.GetSSOTicketTTL(ctx, ticket)
}
