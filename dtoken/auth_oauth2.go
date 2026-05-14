package dtoken

import (
	"context"

	"github.com/Zany2/dtoken-go/core/oauth2"
)

// OAuth2Token handles OAuth2 token request. OAuth2Token 处理 OAuth2 令牌请求。
func (a *Auth) OAuth2Token(ctx context.Context, req *oauth2.TokenRequest, validateUser oauth2.UserValidator) (*oauth2.AccessToken, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.OAuth2Token(ctx, req, validateUser)
}

// RegisterOAuth2Client registers an OAuth2 client. RegisterOAuth2Client 注册 OAuth2 客户端。
func (a *Auth) RegisterOAuth2Client(client *oauth2.Client) error {
	mgr, err := a.requireManager()
	if err != nil {
		return err
	}
	return mgr.RegisterOAuth2Client(client)
}

// ExchangeOAuth2CodeForToken exchanges authorization code for access token. ExchangeOAuth2CodeForToken 使用授权码换取访问令牌。
func (a *Auth) ExchangeOAuth2CodeForToken(ctx context.Context, code, clientID, clientSecret, redirectURI string) (*oauth2.AccessToken, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.ExchangeOAuth2CodeForToken(ctx, code, clientID, clientSecret, redirectURI)
}
