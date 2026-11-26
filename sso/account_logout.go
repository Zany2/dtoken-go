package sso

import (
	"errors"
	"net/http"
)

// ErrAccountLogoutConfig indicates that account logout lacks an explicit center-session revoker. ErrAccountLogoutConfig 表示账号下线缺少显式中心会话撤销函数。
var ErrAccountLogoutConfig = errors.New("sso account logout requires CenterLogout")

// AccountLogoutResult records acknowledged progress of a trusted account-wide logout; completed stages are not rolled back. AccountLogoutResult 记录可信账号下线已确认的进度，已完成阶段不会回滚。
type AccountLogoutResult struct {
	// Stage is validation, center_logout, credentials, notifications, cleanup, or complete. Stage 为校验、中心注销、凭证撤销、通知、清理或完成阶段。
	Stage string `json:"stage"`
	// Complete means all enabled stages succeeded, including delivered or deliberately skipped notifications. Complete 表示全部启用阶段成功，包含成功或主动跳过的通知。
	Complete bool `json:"complete"`
	// CenterRevoked means CenterLogout returned success; it does not independently verify the hook's implementation. CenterRevoked 表示中心撤销钩子返回成功，不独立验证钩子实现。
	CenterRevoked bool `json:"centerRevoked"`
	// CredentialsRevoked means the account credential generation was replaced successfully. CredentialsRevoked 表示账号凭证版本已成功替换。
	CredentialsRevoked bool `json:"credentialsRevoked"`
	// Notifications contains delivery outcomes when EnableSLO is true. Notifications 在启用 EnableSLO 时保存通知结果。
	Notifications *LogoutNotificationReport `json:"notifications,omitempty"`
	// BindingsCleared means callback bindings were cleared after successful notifications; false when SLO is disabled. BindingsCleared 表示通知成功后已清理回调绑定，禁用 SLO 时为 false。
	BindingsCleared bool `json:"bindingsCleared"`
	// LoginID identifies the trusted target and is excluded from JSON. LoginID 标识可信目标，不参与 JSON 序列化。
	LoginID string `json:"-"`
	// Cause preserves the returned error for private diagnostics and is excluded from JSON. Cause 保留返回错误供内部诊断，不参与 JSON 序列化。
	Cause error `json:"-"`
}

// LogoutAccount revokes a trusted target before notifying applications; callers authorize the target and block new login/issuance when disabling an account. LogoutAccount 先撤销可信目标再通知应用；调用方负责目标授权，封禁账号时还须阻止新登录及签发。
func (h *HTTPServer) LogoutAccount(r *http.Request, loginID string) (result *AccountLogoutResult, err error) {
	result = &AccountLogoutResult{LoginID: loginID, Stage: "validation"}
	defer func() { result.Cause = err }()
	if err = h.initializationError(); err != nil {
		return result, err
	}
	if r == nil {
		return result, ErrInvalidRequest
	}
	if loginID == "" {
		return result, ErrUserIDEmpty
	}
	if h.options.CenterLogout == nil {
		return result, ErrAccountLogoutConfig
	}
	ctx := r.Context()
	if err = ctx.Err(); err != nil {
		return result, err
	}

	// Unlike browser logout, notification failures must not retain already revocable center sessions or credentials. 与浏览器退出不同，通知故障不应保留本可撤销的中心会话与凭证。
	result.Stage = "center_logout"
	if err = h.options.CenterLogout(r, loginID); err != nil {
		return result, errors.Join(err, ctx.Err())
	}
	result.CenterRevoked = true
	if err = ctx.Err(); err != nil {
		return result, err
	}

	result.Stage = "credentials"
	if err = h.server.RevokeLoginCredentials(ctx, loginID); err != nil {
		return result, errors.Join(err, ctx.Err())
	}
	result.CredentialsRevoked = true
	if err = ctx.Err(); err != nil {
		return result, err
	}

	if h.options.EnableSLO {
		result.Stage = "notifications"
		result.Notifications, err = h.NotifyLogoutClients(r, loginID)
		if err != nil {
			return result, err
		}
		if err = ctx.Err(); err != nil {
			return result, err
		}

		// Preserve every binding on notification failure, regardless of browser best-effort policy, so callers can retry. 无论浏览器尽力策略如何，通知失败都保留全部绑定，供调用方重试。
		result.Stage = "cleanup"
		if err = h.server.ClearClientSessions(ctx, loginID); err != nil {
			return result, errors.Join(err, ctx.Err())
		}
		result.BindingsCleared = true
		if err = ctx.Err(); err != nil {
			return result, err
		}
	}

	result.Stage, result.Complete = "complete", true
	return result, nil
}
