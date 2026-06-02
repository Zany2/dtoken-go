// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/utils"
)

const refreshTokenByteLength = 32

// RefreshTokenOptions describes refresh-token login options. RefreshTokenOptions 描述刷新令牌登录选项。
type RefreshTokenOptions struct {
	LoginOptions
	RefreshTimeout time.Duration `json:"refreshTimeout"` // RefreshTimeout overrides refresh token timeout. RefreshTimeout 覆盖刷新令牌超时时间。
}

// RefreshTokenPair stores access token and refresh token. RefreshTokenPair 存储访问令牌和刷新令牌。
type RefreshTokenPair struct {
	AccessToken      string `json:"accessToken"`        // AccessToken stores access token value. AccessToken 存储访问令牌值。
	RefreshToken     string `json:"refreshToken"`       // RefreshToken stores refresh token value. RefreshToken 存储刷新令牌值。
	TokenType        string `json:"tokenType"`          // TokenType stores token type. TokenType 存储令牌类型。
	ExpiresIn        int64  `json:"expiresIn"`          // ExpiresIn stores access token ttl seconds. ExpiresIn 存储访问令牌剩余秒数。
	RefreshExpiresIn int64  `json:"refreshExpiresIn"`   // RefreshExpiresIn stores refresh token ttl seconds. RefreshExpiresIn 存储刷新令牌剩余秒数。
	LoginID          string `json:"loginId"`            // LoginID stores subject identifier. LoginID 存储主体标识。
	Device           string `json:"device,omitempty"`   // Device stores device type. Device 存储设备类型。
	DeviceID         string `json:"deviceId,omitempty"` // DeviceID stores concrete device id. DeviceID 存储具体设备 ID。
}

// RefreshTokenInfo stores refresh token metadata. RefreshTokenInfo 存储刷新令牌元数据。
type RefreshTokenInfo struct {
	AuthType    string         `json:"authType"`        // AuthType stores auth namespace. AuthType 存储认证命名空间。
	LoginID     string         `json:"loginId"`         // LoginID stores subject identifier. LoginID 存储主体标识。
	Device      string         `json:"device"`          // Device stores device type. Device 存储设备类型。
	DeviceID    string         `json:"deviceId"`        // DeviceID stores concrete device id. DeviceID 存储具体设备 ID。
	AccessToken string         `json:"accessToken"`     // AccessToken stores related access token. AccessToken 存储关联访问令牌。
	CreateTime  int64          `json:"createTime"`      // CreateTime stores creation timestamp. CreateTime 存储创建时间戳。
	ExpiresIn   int64          `json:"expiresIn"`       // ExpiresIn stores refresh token ttl seconds. ExpiresIn 存储刷新令牌有效秒数。
	AccessTTL   int64          `json:"accessTtl"`       // AccessTTL stores access token ttl seconds. AccessTTL 存储访问令牌有效秒数。
	ActiveTTL   int64          `json:"activeTtl"`       // ActiveTTL stores inactive timeout seconds. ActiveTTL 存储不活跃超时秒数。
	Extra       map[string]any `json:"extra,omitempty"` // Extra stores token extension data. Extra 存储令牌扩展数据。
}

// LoginWithRefreshToken logs in and returns access and refresh tokens. LoginWithRefreshToken 登录并返回访问令牌和刷新令牌。
func (m *Manager) LoginWithRefreshToken(ctx context.Context, loginID string, deviceAndDeviceId ...string) (*RefreshTokenPair, error) {
	device, deviceID := m.getDeviceAndDeviceId(deviceAndDeviceId...)
	return m.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{
		LoginOptions: LoginOptions{
			LoginID:  loginID,
			Device:   device,
			DeviceID: deviceID,
		},
	})
}

// LoginWithRefreshTokenOptions logs in with options and returns token pair. LoginWithRefreshTokenOptions 使用选项登录并返回令牌对。
func (m *Manager) LoginWithRefreshTokenOptions(ctx context.Context, opts RefreshTokenOptions) (*RefreshTokenPair, error) {
	accessToken, err := m.LoginWithOptions(ctx, opts.LoginOptions)
	if err != nil {
		return nil, err
	}
	pair, err := m.issueRefreshToken(ctx, accessToken, opts.RefreshTimeout)
	if err != nil {
		_ = m.Logout(ctx, accessToken)
		return nil, err
	}
	return pair, nil
}

// RefreshToken rotates a refresh token and returns a new token pair. RefreshToken 轮换刷新令牌并返回新的令牌对。
func (m *Manager) RefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenPair, error) {
	info, err := m.getRefreshTokenInfo(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	if info.LoginID == "" {
		return nil, derror.ErrInvalidRefreshToken
	}
	if m.isDisable(ctx, info.LoginID) {
		return nil, derror.ErrAccountDisabled
	}
	if m.isDisableDeviceMatch(ctx, info.LoginID, info.Device, info.DeviceID) {
		return nil, derror.ErrDeviceDisabled
	}

	_ = m.Logout(ctx, info.AccessToken)
	_ = m.storage.Delete(ctx, m.getRefreshTokenKey(refreshToken), m.getTokenRefreshKey(info.AccessToken))

	pair, err := m.LoginWithRefreshTokenOptions(ctx, RefreshTokenOptions{
		LoginOptions: LoginOptions{
			LoginID:       info.LoginID,
			Device:        info.Device,
			DeviceID:      info.DeviceID,
			Timeout:       secondsToDuration(info.AccessTTL),
			ActiveTimeout: secondsToDuration(info.ActiveTTL),
			Extra:         info.Extra,
		},
		RefreshTimeout: secondsToDuration(info.ExpiresIn),
	})
	if err != nil {
		return nil, err
	}
	return pair, nil
}

// RevokeRefreshToken revokes a refresh token and its related access token. RevokeRefreshToken 撤销刷新令牌及其关联访问令牌。
func (m *Manager) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	info, err := m.getRefreshTokenInfo(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, derror.ErrInvalidRefreshToken) {
			return nil
		}
		return err
	}
	if info.AccessToken != "" {
		_ = m.Logout(ctx, info.AccessToken)
	}
	return m.storage.Delete(ctx, m.getRefreshTokenKey(refreshToken), m.getTokenRefreshKey(info.AccessToken))
}

// GetRefreshTokenTTL returns refresh token remaining lifetime seconds. GetRefreshTokenTTL 返回刷新令牌剩余有效秒数。
func (m *Manager) GetRefreshTokenTTL(ctx context.Context, refreshToken string) (int64, error) {
	if refreshToken == "" {
		return 0, derror.ErrInvalidRefreshToken
	}
	ttl, err := m.storage.TTL(ctx, m.getRefreshTokenKey(refreshToken))
	if err != nil {
		return 0, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return normalizeTTLSeconds(ttl), nil
}

// issueRefreshToken creates a refresh token for an existing access token. issueRefreshToken 为已有访问令牌创建刷新令牌。
func (m *Manager) issueRefreshToken(ctx context.Context, accessToken string, refreshTimeout time.Duration) (*RefreshTokenPair, error) {
	tokenInfo, err := m.getTokenInfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	accessTTL, err := m.GetTokenTTL(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}
	expiration := m.resolveRefreshTokenExpiration(refreshTimeout)
	info := RefreshTokenInfo{
		AuthType:    m.config.AuthType,
		LoginID:     tokenInfo.LoginID,
		Device:      tokenInfo.Device,
		DeviceID:    tokenInfo.DeviceId,
		AccessToken: accessToken,
		CreateTime:  time.Now().Unix(),
		ExpiresIn:   m.timeoutToSeconds(expiration),
		AccessTTL:   tokenInfo.Timeout,
		ActiveTTL:   tokenInfo.ActiveTimeout,
		Extra:       tokenInfo.Extra,
	}
	if err = m.saveToStorage(ctx, m.getRefreshTokenKey(refreshToken), info, expiration); err != nil {
		return nil, err
	}
	if err = m.storage.Set(ctx, m.getTokenRefreshKey(accessToken), refreshToken, m.resolveTokenExpiration(tokenInfo)); err != nil {
		_ = m.storage.Delete(ctx, m.getRefreshTokenKey(refreshToken))
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	refreshTTL, err := m.GetRefreshTokenTTL(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	return &RefreshTokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        accessTTL,
		RefreshExpiresIn: refreshTTL,
		LoginID:          tokenInfo.LoginID,
		Device:           tokenInfo.Device,
		DeviceID:         tokenInfo.DeviceId,
	}, nil
}

// getRefreshTokenInfo loads refresh token metadata. getRefreshTokenInfo 加载刷新令牌元数据。
func (m *Manager) getRefreshTokenInfo(ctx context.Context, refreshToken string) (*RefreshTokenInfo, error) {
	if refreshToken == "" {
		return nil, derror.ErrInvalidRefreshToken
	}
	data, err := m.storage.Get(ctx, m.getRefreshTokenKey(refreshToken))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, derror.ErrInvalidRefreshToken
	}
	rawData, err := utils.ToBytes(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}
	var info RefreshTokenInfo
	if err = m.serializer.Decode(rawData, &info); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrSerializeFailed, err)
	}
	return &info, nil
}

// cleanRefreshTokenByAccessToken deletes refresh token linked to access token. cleanRefreshTokenByAccessToken 删除访问令牌关联的刷新令牌。
func (m *Manager) cleanRefreshTokenByAccessToken(ctx context.Context, accessToken string) error {
	if accessToken == "" {
		return nil
	}
	data, err := m.storage.Get(ctx, m.getTokenRefreshKey(accessToken))
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil
	}
	refreshBytes, err := utils.ToBytes(data)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}
	refreshToken := string(refreshBytes)
	return m.storage.Delete(ctx, m.getRefreshTokenKey(refreshToken), m.getTokenRefreshKey(accessToken))
}

// resolveRefreshTokenExpiration resolves refresh token ttl. resolveRefreshTokenExpiration 解析刷新令牌有效期。
func (m *Manager) resolveRefreshTokenExpiration(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}
	if timeout < 0 || m.config.RefreshTokenTimeout == config.NoLimit {
		return 0
	}
	return time.Duration(m.config.RefreshTokenTimeout) * time.Second
}

// generateRefreshToken generates a random refresh token. generateRefreshToken 生成随机刷新令牌。
func generateRefreshToken() (string, error) {
	bytes := make([]byte, refreshTokenByteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// secondsToDuration converts seconds to duration. secondsToDuration 将秒转换为时长。
func secondsToDuration(seconds int64) time.Duration {
	if seconds == config.NoLimit {
		return -1
	}
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
