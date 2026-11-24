package sso

import (
	"context"
	"fmt"
)

// RevokeLoginCredentials invalidates previously issued versioned credentials for an account across all clients. RevokeLoginCredentials 使账号在所有客户端此前签发的带版本凭证失效。
func (s *Server) RevokeLoginCredentials(ctx context.Context, loginID string) error {
	if loginID == "" {
		return ErrUserIDEmpty
	}
	version, err := s.generateRandomValue(32, "SSO logout version")
	if err != nil {
		return err
	}

	// Keep the marker persistent so old credentials cannot become valid again after marker expiry. 永久保留标记，避免旧凭证因标记过期重新生效。
	if err := s.storage.Set(ctx, s.getLogoutVersionKey(loginID), version, 0); err != nil {
		return fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	return nil
}

// getLogoutVersionKey isolates account revocation markers within the server namespace. getLogoutVersionKey 在服务端命名空间内隔离账号撤销标记。
func (s *Server) getLogoutVersionKey(loginID string) string {
	return s.keyPrefix + s.authType + "sso:logout-version:" + loginID
}

// getLogoutVersion reads the current account version without creating state on first login. getLogoutVersion 读取账号当前版本，首次登录不创建额外状态。
func (s *Server) getLogoutVersion(ctx context.Context, loginID string) (string, error) {
	value, err := s.storage.Get(ctx, s.getLogoutVersionKey(loginID))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	if value == nil {
		return "initial", nil
	}
	data, err := toBytes(value)
	if err != nil || len(data) == 0 {
		return "", fmt.Errorf("%w: invalid SSO logout version", ErrTypeConvert)
	}
	return string(data), nil
}

// checkLogoutVersion rejects revoked credentials while preserving records issued before versioning. checkLogoutVersion 拒绝已撤销凭证，并兼容版本机制引入前签发的记录。
func (s *Server) checkLogoutVersion(ctx context.Context, loginID, version string, invalid error) error {
	if version == "" {
		return nil
	}
	current, err := s.getLogoutVersion(ctx, loginID)
	if err != nil {
		return err
	}
	if current != version {
		return invalid
	}
	return nil
}
