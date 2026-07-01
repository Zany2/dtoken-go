// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Zany2/dtoken-go/core/adapter"
	"github.com/Zany2/dtoken-go/core/config"
	"github.com/Zany2/dtoken-go/core/derror"
	"github.com/Zany2/dtoken-go/core/listener"
	"github.com/Zany2/dtoken-go/core/utils"
)

// refreshTokenByteLength defines random byte length before hex encoding. refreshTokenByteLength 定义十六进制编码前的随机字节长度。
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
	pair, err := m.loginWithRefreshTokenOptions(ctx, opts)
	if err != nil {
		return nil, err
	}
	m.triggerRefreshTokenEvent(listener.EventRefreshTokenCreate, pair, listener.ActionCreate)
	return pair, nil
}

// loginWithRefreshTokenOptions logs in and issues refresh token pair. loginWithRefreshTokenOptions 登录并签发刷新令牌对。
func (m *Manager) loginWithRefreshTokenOptions(ctx context.Context, opts RefreshTokenOptions) (*RefreshTokenPair, error) {
	return m.loginWithRefreshTokenOptionsInternal(ctx, opts, loginInternalOptions{})
}

// loginWithRefreshTokenOptionsInternal logs in and issues a refresh token with internal login controls. loginWithRefreshTokenOptionsInternal 使用内部登录控制参数登录并签发刷新令牌。
func (m *Manager) loginWithRefreshTokenOptionsInternal(ctx context.Context, opts RefreshTokenOptions, internal loginInternalOptions) (*RefreshTokenPair, error) {
	// Disable sharing so refresh-token login always owns a dedicated access token. 禁用共享，确保刷新令牌登录独占新的访问 Token。
	isShare := false
	opts.LoginOptions.IsShare = &isShare
	accessToken, err := m.loginWithOptionsInternal(ctx, opts.LoginOptions, internal)
	if err != nil {
		return nil, err
	}
	// Remove stale refresh mapping before issuing the new pair. 签发新令牌对前清理旧的刷新令牌映射。
	if err = m.cleanRefreshTokenByAccessToken(ctx, accessToken); err != nil {
		_ = m.Logout(ctx, accessToken)
		return nil, err
	}
	// Roll back access token when refresh token persistence fails. 刷新令牌持久化失败时回滚访问 Token。
	pair, err := m.issueRefreshToken(ctx, accessToken, opts.RefreshTimeout)
	if err != nil {
		_ = m.Logout(ctx, accessToken)
		return nil, err
	}
	return pair, nil
}

// RefreshToken rotates a refresh token and returns a new token pair. RefreshToken 轮换刷新令牌并返回新的令牌对。
func (m *Manager) RefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenPair, error) {
	// Load refresh token metadata before consuming it. 消费刷新令牌前先加载其元数据。
	info, err := m.getRefreshTokenInfo(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	// Reject malformed refresh token records. 拒绝不完整的刷新令牌记录。
	if info.LoginID == "" {
		return nil, derror.ErrInvalidRefreshToken
	}
	// Recheck account and device status at rotation time. 轮换时重新检查账号和设备状态。
	if m.isDisable(ctx, info.LoginID) {
		return nil, derror.ErrAccountDisabled
	}
	if m.isDisableDeviceMatch(ctx, info.LoginID, info.Device, info.DeviceID) {
		return nil, derror.ErrDeviceDisabled
	}

	// Atomically consume refresh token to prevent concurrent replay 原子消费刷新令牌，防止并发重放。
	atomicStorage, ok := m.storage.(adapter.AtomicStorage)
	if !ok {
		return nil, fmt.Errorf("%w: refresh token rotation requires atomic storage", derror.ErrStorageUnavailable)
	}
	existing, delErr := atomicStorage.GetAndDeleteMany(ctx, m.getRefreshTokenKey(refreshToken), m.getTokenRefreshKey(info.AccessToken))
	if delErr != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, delErr)
	}
	if existing == nil {
		return nil, derror.ErrInvalidRefreshToken
	}

	// Create the replacement pair without applying normal concurrency eviction again. 创建替换令牌对时跳过常规并发顶替处理。
	pair, err := m.loginWithRefreshTokenOptionsInternal(ctx, RefreshTokenOptions{
		LoginOptions: LoginOptions{
			LoginID:       info.LoginID,
			Device:        info.Device,
			DeviceID:      info.DeviceID,
			Timeout:       secondsToDuration(info.AccessTTL),
			ActiveTimeout: secondsToDuration(info.ActiveTTL),
			Extra:         info.Extra,
		},
		RefreshTimeout: secondsToDuration(info.ExpiresIn),
	}, loginInternalOptions{skipConcurrencyControl: true})
	if err != nil {
		return nil, err
	}
	// Retire old access token after the replacement pair is available. 新令牌对可用后再下线旧访问 Token。
	if err = m.logoutRotatedAccessToken(ctx, info.AccessToken, pair); err != nil {
		return nil, err
	}
	m.triggerRefreshTokenEvent(listener.EventRefreshTokenRotate, pair, listener.ActionRotate)
	return pair, nil
}

// RevokeRefreshToken revokes a refresh token and its related access token. RevokeRefreshToken 撤销刷新令牌及其关联访问令牌。
func (m *Manager) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	// Treat empty revoke requests as idempotent no-op. 空撤销请求按幂等空操作处理。
	if refreshToken == "" {
		return nil
	}
	// Load refresh metadata and ignore already-invalid tokens. 加载刷新元数据，并忽略已失效的令牌。
	info, err := m.getRefreshTokenInfo(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, derror.ErrInvalidRefreshToken) {
			return nil
		}
		return err
	}
	if info.AccessToken != "" {
		// Revoke the related access token first when it is still active. 关联访问 Token 仍有效时优先撤销它。
		if err = m.Logout(ctx, info.AccessToken); err != nil && !isTokenInactiveError(err) {
			return err
		}
	}
	// Delete both refresh token and reverse lookup keys. 同时删除刷新令牌键和反向索引键。
	err = m.storage.Delete(ctx, m.getRefreshTokenKey(refreshToken), m.getTokenRefreshKey(info.AccessToken))
	if err == nil {
		m.triggerEvent(listener.EventRefreshTokenRevoke, info.LoginID, info.Device, info.DeviceID, info.AccessToken, map[string]any{
			listener.ExtraKeyAction:       listener.ActionRevoke,
			listener.ExtraKeyRefreshToken: refreshToken,
		})
	}
	return err
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
	// Load access token metadata used to bind the refresh token. 加载用于绑定刷新令牌的访问 Token 元数据。
	tokenInfo, err := m.getTokenInfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	// Read access token TTL for response payload. 读取访问 Token TTL 用于响应载荷。
	accessTTL, err := m.GetTokenTTL(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	// Generate an opaque refresh token value. 生成不透明刷新令牌值。
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}
	// Persist refresh metadata with the configured lifetime. 按配置生命周期持久化刷新令牌元数据。
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
	saved, err := m.saveToStorageIfAbsent(ctx, m.getRefreshTokenKey(refreshToken), info, expiration)
	if err != nil {
		return nil, err
	}
	if !saved {
		return nil, fmt.Errorf("%w: refresh token already exists", derror.ErrStorageUnavailable)
	}
	// Store reverse lookup from access token to refresh token. 存储访问 Token 到刷新令牌的反向索引。
	if err = m.storage.Set(ctx, m.getTokenRefreshKey(accessToken), refreshToken, m.resolveTokenExpiration(tokenInfo)); err != nil {
		_ = m.storage.Delete(ctx, m.getRefreshTokenKey(refreshToken))
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}

	// Read final refresh TTL because storage may normalize expiration values. 读取最终刷新 TTL，因为存储层可能会规范化过期值。
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

// logoutRotatedAccessToken logs out old access token after rotation. logoutRotatedAccessToken 在轮换后登出旧访问令牌。
func (m *Manager) logoutRotatedAccessToken(ctx context.Context, oldAccessToken string, pair *RefreshTokenPair) error {
	if oldAccessToken == "" {
		return nil
	}
	if err := m.Logout(ctx, oldAccessToken); err != nil {
		if isTokenInactiveError(err) {
			return nil
		}
		if pair != nil {
			_ = m.cleanRefreshTokenByAccessToken(ctx, pair.AccessToken)
			_ = m.Logout(ctx, pair.AccessToken)
		}
		return err
	}
	return nil
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

// triggerRefreshTokenEvent emits refresh-token lifecycle events. triggerRefreshTokenEvent 触发刷新令牌生命周期事件。
func (m *Manager) triggerRefreshTokenEvent(event listener.Event, pair *RefreshTokenPair, action string) {
	if pair == nil {
		return
	}
	m.triggerEvent(event, pair.LoginID, pair.Device, pair.DeviceID, pair.AccessToken, map[string]any{
		listener.ExtraKeyAction:       action,
		listener.ExtraKeyTokenType:    pair.TokenType,
		listener.ExtraKeyRefreshToken: pair.RefreshToken,
		listener.ExtraKeyTTL:          pair.RefreshExpiresIn,
	})
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
