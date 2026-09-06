// @Author daixk 2025/12/22 15:56:00
package manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
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
	// LoginOptions embeds access-token login options LoginOptions 嵌入访问令牌登录选项
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

// refreshTokenRecord extends persisted metadata without changing the public refresh info shape. refreshTokenRecord 扩展持久化元数据，但不改变公开刷新信息结构。
type refreshTokenRecord struct {
	RefreshTokenInfo `msgpack:",inline"` // RefreshTokenInfo keeps existing fields flat for compatible codecs. RefreshTokenInfo 保持已有字段平铺，以兼容现有编解码格式。
	TerminalExtra    map[string]any      `json:"terminalExtra,omitempty" msgpack:",omitempty"` // TerminalExtra survives access-token and session expiration. TerminalExtra 在访问令牌及 Session 过期后仍可保留。
	AccessID         string              `json:"accessId,omitempty" msgpack:",omitempty"`      // AccessID binds cleanup to the original access lifecycle. AccessID 将清理绑定到原访问生命周期。
}

// LoginWithRefreshToken logs in and returns access and refresh tokens. LoginWithRefreshToken 登录并返回访问令牌和刷新令牌。
// Device arguments are optional and limited to device type followed by device ID. 设备参数可省略，最多依次提供设备类型和设备 ID。
func (m *Manager) LoginWithRefreshToken(ctx context.Context, loginID string, deviceAndDeviceID ...string) (*RefreshTokenPair, error) {
	// Preserve empty-account error precedence before validating device arguments. 校验设备参数前保留空账号错误的优先级。
	if loginID == "" {
		return nil, derror.ErrIDIsEmpty
	}
	if len(deviceAndDeviceID) > 2 {
		return nil, derror.ErrInvalidParam
	}

	device, deviceID := m.getDeviceAndDeviceID(deviceAndDeviceID...)
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
	// Choose the original identity before login callbacks can replace the token mapping. 登录回调可能替换 Token 映射，因此提前确定原始身份。
	if internal.accessID == "" {
		internal.accessID = rand.Text()
	}

	// Disable sharing so refresh-token login always owns a dedicated access token. 禁用共享，确保刷新令牌登录独占新的访问 Token。
	isShare := false
	opts.LoginOptions.IsShare = &isShare
	accessToken, err := m.loginWithOptionsInternal(ctx, opts.LoginOptions, internal)
	if err != nil {
		return nil, err
	}

	// Preserve caller identity rather than adopting whatever currently occupies the token key. 保留调用方的原始身份，不接管当前占据 Token 键的其他登录。
	expected := refreshTokenRecord{
		RefreshTokenInfo: RefreshTokenInfo{
			LoginID: opts.LoginID, AccessToken: accessToken,
			Device: strings.TrimSpace(opts.Device), DeviceID: strings.TrimSpace(opts.DeviceID),
		},
		AccessID: internal.accessID,
	}

	// Keep token-pair creation all-or-nothing for callers. 对调用方保持令牌对创建的整体一致性。
	pair, err := m.issueRefreshToken(ctx, &expected, opts.RefreshTimeout, opts.TerminalExtra)
	if err != nil {
		_ = m.removeRefreshAccessToken(ctx, &expected)
		return nil, err
	}
	return pair, nil
}

// RefreshToken rotates a refresh token and returns a new token pair. RefreshToken 轮换刷新令牌并返回新的令牌对。
func (m *Manager) RefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenPair, error) {
	// Load refresh metadata once to resolve the account lock. 先加载刷新令牌元数据以确定账号锁。
	info, err := m.getRefreshTokenInfo(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// Reject malformed refresh token records. 拒绝不完整的刷新令牌记录。
	if info.LoginID == "" || info.AccessToken == "" {
		return nil, derror.ErrInvalidRefreshToken
	}
	loginID := info.LoginID

	// Serialize local consumption with refresh revocation and account lifecycle writes. 将本地消费与刷新令牌撤销、账号生命周期写操作串行化。
	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	// Reload under the account lock so stale metadata cannot drive rotation. 在账号锁内重新加载，避免陈旧元数据驱动轮换。
	info, err = m.getRefreshTokenInfo(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	if info.LoginID != loginID || info.AccessToken == "" {
		return nil, derror.ErrInvalidRefreshToken
	}

	// Recheck account and device status immediately before consumption. 消费前重新检查账号和设备状态。
	if err = m.checkLoginDisableState(ctx, info.LoginID, info.Device, info.DeviceID); err != nil {
		return nil, err
	}

	// Prefer atomic consumption and fall back to the account lock for basic storage implementations. 优先原子消费；基础存储实现回退为账号锁保护的读取删除。
	consumedInfo, err := m.consumeRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	if consumedInfo.LoginID != info.LoginID ||
		consumedInfo.AccessToken != info.AccessToken ||
		consumedInfo.AccessID != info.AccessID ||
		consumedInfo.CreateTime != info.CreateTime {
		return nil, derror.ErrInvalidRefreshToken
	}
	info = consumedInfo
	if err = m.removeRefreshTokenReverseLookup(ctx, info.AccessToken, refreshToken); err != nil {
		return nil, err
	}

	// Release the account lock before the replacement login acquires it. 替换登录再次获取账号锁前先释放当前锁。
	unlock()
	unlock = func() {}

	// Create the replacement pair without applying normal concurrency eviction again. 创建替换令牌对时跳过常规并发顶替处理。
	replacementAccessID := rand.Text()
	pair, err := m.loginWithRefreshTokenOptionsInternal(ctx, RefreshTokenOptions{
		LoginOptions: LoginOptions{
			LoginID:       info.LoginID,
			Device:        info.Device,
			DeviceID:      info.DeviceID,
			Timeout:       secondsToDuration(info.AccessTTL),
			ActiveTimeout: secondsToDuration(info.ActiveTTL),
			Extra:         info.Extra,
			TerminalExtra: info.TerminalExtra,
		},
		RefreshTimeout: secondsToDuration(info.ExpiresIn),
	}, loginInternalOptions{skipConcurrencyControl: true, accessID: replacementAccessID})
	if err != nil {
		return nil, err
	}

	// Retire old access token after the replacement pair is available. 新令牌对可用后再下线旧访问 Token。
	if err = m.logoutRotatedAccessToken(ctx, info, pair, replacementAccessID); err != nil {
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

	// Serialize local revocation with refresh consumption and account lifecycle writes. 将本地撤销与刷新令牌消费、账号生命周期写操作串行化。
	loginID := info.LoginID
	unlock := m.lockLoginWrite(loginID)
	defer func() { unlock() }()

	// Reload and consume first so a concurrent refresh cannot succeed after revocation wins. 先重新加载并消费，确保撤销先完成时并发刷新无法成功。
	info, err = m.getRefreshTokenInfo(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, derror.ErrInvalidRefreshToken) {
			return nil
		}
		return err
	}
	if info.LoginID != loginID {
		return derror.ErrInvalidRefreshToken
	}
	info, err = m.consumeRefreshToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, derror.ErrInvalidRefreshToken) {
			return nil
		}
		return err
	}

	// Delete only the reverse lookup still owned by the consumed refresh token. 仅删除仍属于已消费刷新令牌的反向索引。
	if err = m.removeRefreshTokenReverseLookup(ctx, info.AccessToken, refreshToken); err != nil {
		return err
	}

	// Release before access-token cleanup re-enters the same account lock. 清理访问 Token 再次进入账号锁前先释放。
	unlock()
	unlock = func() {}
	if info.AccessToken != "" {
		// Remove only an access lifecycle that can still be identified. 仅移除仍能确认身份的访问生命周期。
		if err = m.removeRefreshAccessToken(ctx, info); err != nil {
			return err
		}
	}

	m.triggerEvent(listener.EventRefreshTokenRevoke, info.LoginID, info.Device, info.DeviceID, info.AccessToken, map[string]any{
		listener.ExtraKeyAction:       listener.ActionRevoke,
		listener.ExtraKeyRefreshToken: refreshToken,
	})
	return nil
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
func (m *Manager) issueRefreshToken(ctx context.Context, expected *refreshTokenRecord, refreshTimeout time.Duration, terminalExtra map[string]any) (*RefreshTokenPair, error) {
	// Serialize binding checks and issuance with account lifecycle writes. 将绑定校验、签发与账号生命周期写入串行化。
	unlock := m.lockLoginWrite(expected.LoginID)
	defer unlock()
	accessToken := expected.AccessToken

	// Load access token metadata used to bind the refresh token. 加载用于绑定刷新令牌的访问 Token 元数据。
	tokenInfo, err := m.getTokenRecord(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	if tokenInfo.LoginID != expected.LoginID || expected.AccessID == "" || tokenInfo.AccessID != expected.AccessID {
		return nil, derror.ErrInvalidToken
	}

	// Callbacks may have removed the session or disabled the original login. 回调可能已移除 Session 或封禁原登录。
	if err = m.checkLoginDisableState(ctx, tokenInfo.LoginID, tokenInfo.Device, tokenInfo.DeviceID); err != nil {
		return nil, err
	}
	alive, err := m.checkTerminalTokenStructurallyAliveWithContext(ctx, accessToken, &tokenInfo.TokenInfo, nil)
	if err != nil {
		return nil, err
	}
	if !alive {
		return nil, derror.ErrInvalidToken
	}

	// Only a verified original login may remove its stale reverse mapping. 只有确认原登录身份后才能清理其陈旧反向映射。
	if err = m.cleanRefreshTokenByAccessToken(ctx, accessToken); err != nil {
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
		DeviceID:    tokenInfo.DeviceID,
		AccessToken: accessToken,
		CreateTime:  time.Now().Unix(),
		ExpiresIn:   m.timeoutToSeconds(expiration),
		AccessTTL:   tokenInfo.Timeout,
		ActiveTTL:   tokenInfo.ActiveTimeout,
		Extra:       tokenInfo.Extra,
	}

	// Persist terminal data with the refresh lifetime instead of relying on the shorter-lived session. 按刷新令牌生命周期保存终端数据，不依赖有效期较短的 Session。
	record := refreshTokenRecord{RefreshTokenInfo: info, TerminalExtra: terminalExtra, AccessID: tokenInfo.AccessID}
	saved, err := m.saveToStorageIfAbsent(ctx, m.getRefreshTokenKey(refreshToken), record, expiration)
	if err != nil {
		return nil, err
	}
	if !saved {
		return nil, fmt.Errorf("%w: refresh token already exists", derror.ErrStorageUnavailable)
	}

	// Store reverse lookup no longer than either side. 反向索引有效期不超过访问令牌或刷新令牌任一侧。
	reverseExpiration := m.resolveTokenExpiration(&tokenInfo.TokenInfo)
	if expiration > 0 && (reverseExpiration <= 0 || reverseExpiration > expiration) {
		reverseExpiration = expiration
	}
	if err = m.storage.Set(ctx, m.getTokenRefreshKey(accessToken), refreshToken, reverseExpiration); err != nil {
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
		DeviceID:         tokenInfo.DeviceID,
	}, nil
}

// getRefreshTokenInfo loads refresh token metadata. getRefreshTokenInfo 加载刷新令牌元数据。
func (m *Manager) getRefreshTokenInfo(ctx context.Context, refreshToken string) (*refreshTokenRecord, error) {
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
	return m.decodeRefreshTokenInfo(data)
}

// consumeRefreshToken removes and returns refresh metadata once. consumeRefreshToken 一次性删除并返回刷新令牌元数据。
func (m *Manager) consumeRefreshToken(ctx context.Context, refreshToken string) (*refreshTokenRecord, error) {
	if refreshToken == "" {
		return nil, derror.ErrInvalidRefreshToken
	}

	// Use storage-native atomic consumption when available. 存储支持时使用原子消费。
	key := m.getRefreshTokenKey(refreshToken)
	if atomicStorage, ok := m.storage.(adapter.AtomicStorage); ok {
		data, err := atomicStorage.GetAndDelete(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		if data == nil {
			return nil, derror.ErrInvalidRefreshToken
		}
		return m.decodeRefreshTokenInfo(data)
	}

	// Caller holds the account lock for the ordinary Get/Delete fallback. 普通 Get/Delete 回退由调用方持有账号锁保护。
	data, err := m.storage.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil, derror.ErrInvalidRefreshToken
	}
	if err = m.storage.Delete(ctx, key); err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return m.decodeRefreshTokenInfo(data)
}

// decodeRefreshTokenInfo decodes stored refresh metadata. decodeRefreshTokenInfo 解码存储中的刷新令牌元数据。
func (m *Manager) decodeRefreshTokenInfo(data any) (*refreshTokenRecord, error) {
	rawData, err := utils.ToBytes(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", derror.ErrTypeConvert, err)
	}

	// Legacy records decode without terminal data or lifecycle identity. 旧记录解码后可缺少终端数据和生命周期标识。
	var info refreshTokenRecord
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
		// Drop corrupt reverse metadata and keep cleanup idempotent. 删除损坏的反向元数据并保持清理幂等。
		if deleteErr := m.storage.Delete(ctx, m.getTokenRefreshKey(accessToken)); deleteErr != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, deleteErr)
		}
		return nil
	}
	refreshToken := string(refreshBytes)
	if refreshToken == "" {
		if err := m.storage.Delete(ctx, m.getTokenRefreshKey(accessToken)); err != nil {
			return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
		}
		return nil
	}
	if err := m.storage.Delete(ctx, m.getRefreshTokenKey(refreshToken), m.getTokenRefreshKey(accessToken)); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

// removeRefreshTokenReverseLookup deletes a matching reverse binding while the caller holds the account lock. removeRefreshTokenReverseLookup 在调用方持有账号锁时删除匹配的反向绑定。
func (m *Manager) removeRefreshTokenReverseLookup(ctx context.Context, accessToken, refreshToken string) error {
	if accessToken == "" {
		return nil
	}
	key := m.getTokenRefreshKey(accessToken)
	data, err := m.storage.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	if data == nil {
		return nil
	}
	value, err := utils.ToBytes(data)
	if err != nil || string(value) != refreshToken {
		// Unverifiable or newer bindings do not belong to this cleanup. 无法确认或较新的绑定不属于本次清理。
		return nil
	}
	if err = m.storage.Delete(ctx, key); err != nil {
		return fmt.Errorf("%w: %v", derror.ErrStorageUnavailable, err)
	}
	return nil
}

// removeRefreshAccessToken removes the access terminal bound to refresh metadata. removeRefreshAccessToken 移除刷新元数据绑定的访问终端。
func (m *Manager) removeRefreshAccessToken(ctx context.Context, info *refreshTokenRecord) error {
	if info == nil || info.LoginID == "" || info.AccessToken == "" {
		return nil
	}

	// Build detached terminal metadata so cleanup still invalidates the access token when the session list is stale. 构建脱离会话的终端元数据，确保 Session 终端列表过期时仍能使访问 Token 失效。
	detached := TerminalInfo{
		Token:    info.AccessToken,
		LoginID:  info.LoginID,
		Device:   info.Device,
		DeviceID: info.DeviceID,
	}
	return m.logoutTerminalsIf(ctx, info.LoginID, func() (bool, error) {
		current, err := m.getTokenRecord(ctx, info.AccessToken)
		if err != nil {
			if isTokenInactiveError(err) {
				return false, nil
			}
			return false, err
		}
		if current.LoginID != info.LoginID || current.AccessID != info.AccessID {
			return false, nil
		}

		// Legacy-to-legacy matching is best effort; new-format logins never match missing identities. 旧记录之间尽力匹配；缺失标识时绝不匹配新格式登录。
		if info.AccessID == "" {
			return current.Device == info.Device && current.DeviceID == info.DeviceID && current.CreateTime <= info.CreateTime, nil
		}
		return true, nil
	}, func(sess *Session) []TerminalInfo {
		if terminal, ok := sess.removeTerminalByToken(info.AccessToken); ok {
			return []TerminalInfo{terminal}
		}
		return nil
	}, detached)
}

// logoutRotatedAccessToken logs out old access token after rotation. logoutRotatedAccessToken 在轮换后登出旧访问令牌。
func (m *Manager) logoutRotatedAccessToken(ctx context.Context, oldInfo *refreshTokenRecord, pair *RefreshTokenPair, replacementAccessID string) error {
	if oldInfo == nil || oldInfo.AccessToken == "" {
		return nil
	}
	if err := m.removeRefreshAccessToken(ctx, oldInfo); err != nil {
		if pair != nil {
			// Failure cleanup owns only the replacement lifecycle, not a later login using its token text. 失败清理仅拥有替代生命周期，不拥有后续复用其 Token 文本的登录。
			_ = m.removeRefreshAccessToken(ctx, &refreshTokenRecord{
				RefreshTokenInfo: RefreshTokenInfo{
					LoginID: pair.LoginID, AccessToken: pair.AccessToken,
					Device: pair.Device, DeviceID: pair.DeviceID,
				},
				AccessID: replacementAccessID,
			})
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
