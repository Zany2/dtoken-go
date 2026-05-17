// @Author daixk 2025/12/22 15:56:00
package dtoken

import (
	"context"

	"github.com/Zany2/dtoken-go/core/manager"
)

// GetSession gets session by login id. GetSession 根据登录 ID 获取会话。
func (a *Auth) GetSession(ctx context.Context, loginID string) (*manager.Session, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetSession(ctx, loginID)
}

// GetSessionByToken gets session by token. GetSessionByToken 根据 Token 获取会话。
func (a *Auth) GetSessionByToken(ctx context.Context, token string) (*manager.Session, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetSessionByToken(ctx, token)
}

// GetTerminalInfoByToken gets terminal info by token. GetTerminalInfoByToken 根据 Token 获取终端信息。
func (a *Auth) GetTerminalInfoByToken(ctx context.Context, token string) (*manager.TerminalInfo, error) {
	mgr, err := a.requireManager()
	if err != nil {
		return nil, err
	}
	return mgr.GetTerminalInfoByToken(ctx, token)
}
