package sso

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

// LogoutCallbackStatus identifies the final delivery state of one client. LogoutCallbackStatus 标识单个客户端的最终通知状态。
type LogoutCallbackStatus string

const (
	// LogoutCallbackDelivered means a callback acknowledged logout. LogoutCallbackDelivered 表示回调已确认注销。
	LogoutCallbackDelivered LogoutCallbackStatus = "delivered"
	// LogoutCallbackSkipped means no callback is configured or the client was removed. LogoutCallbackSkipped 表示无回调地址或客户端登记已删除。
	LogoutCallbackSkipped LogoutCallbackStatus = "skipped"
	// LogoutCallbackFailed means delivery failed after its permitted attempts. LogoutCallbackFailed 表示允许的尝试结束后通知失败。
	LogoutCallbackFailed LogoutCallbackStatus = "failed"
	// LogoutCallbackCanceled means the parent request ended before delivery completed. LogoutCallbackCanceled 表示通知完成前父请求已终止。
	LogoutCallbackCanceled LogoutCallbackStatus = "canceled"
)

// LogoutCallbackResult exposes one target's outcome without callback URLs or raw transport errors. LogoutCallbackResult 暴露单目标结果，不输出回调地址或原始传输错误。
type LogoutCallbackResult struct {
	// ClientID identifies the registered application. ClientID 标识登记的应用。
	ClientID string `json:"clientId"`
	// Status identifies delivery, failure, cancellation, or deliberate omission. Status 标识成功、失败、取消或主动跳过。
	Status LogoutCallbackStatus `json:"status"`
	// Attempts counts delivery attempts including registration validation, excluding queue waits. Attempts 记录包含登记校验的通知尝试次数，不含排队等待。
	Attempts int `json:"attempts"`
	// HTTPStatus is the final attempt's HTTP status, or zero when no response was received. HTTPStatus 为最后一次尝试的 HTTP 状态，无响应时为零。
	HTTPStatus int `json:"httpStatus,omitempty"`
	// Reason is a stable category such as transport, timeout, http_status, acknowledgement, configuration, storage, canceled, no_callback, or unregistered. Reason 为稳定的失败或跳过分类。
	Reason string `json:"reason,omitempty"`
	// Cause retains the underlying error for application diagnostics and is never serialized. Cause 保留应用诊断所需的原始错误，不参与序列化。
	Cause error `json:"-"`
}

// LogoutNotificationReport contains final results in client-ID order. LogoutNotificationReport 按客户端 ID 顺序保存最终结果。
type LogoutNotificationReport struct {
	// Complete means every target was delivered or deliberately skipped, not that center logout completed. Complete 表示所有目标均通知成功或主动跳过，不代表中心注销完成。
	Complete bool `json:"complete"`
	// Clients contains one result per stored client binding. Clients 包含每条已存储客户端绑定的结果。
	Clients []LogoutCallbackResult `json:"clients"`
}

// LogoutResult describes the final stage reached by an authenticated HTTP logout. LogoutResult 描述已认证 HTTP 注销最终到达的阶段。
type LogoutResult struct {
	// Result retains the legacy "ok" success marker and is empty on failure. Result 保留旧版成功标记 "ok"，失败时为空。
	Result string `json:"result,omitempty"`
	// Stage is notifications, center_logout, credentials, cleanup, or complete. Stage 为通知、中心会话注销、凭证撤销、清理或完成阶段。
	Stage string `json:"stage"`
	// Notifications contains callback outcomes when SLO is enabled. Notifications 在启用单点注销时包含通知结果。
	Notifications *LogoutNotificationReport `json:"notifications,omitempty"`
	// LoginID is available to trusted observers only. LoginID 仅供可信观察钩子使用。
	LoginID string `json:"-"`
	// Cause retains the final lifecycle failure and is not serialized. Cause 保留最终生命周期错误，不参与序列化。
	Cause error `json:"-"`
}

// LogoutResultFunc observes completed or failed authenticated logout attempts synchronously. LogoutResultFunc 同步观察已认证注销尝试的成功或失败结果。
type LogoutResultFunc func(r *http.Request, result LogoutResult)

// NotifyLogoutClients notifies a trusted caller's subject without revoking center state; authorization belongs to the caller. NotifyLogoutClients 为可信调用方指定的主体发送通知，不撤销中心状态，调用方负责授权。
func (h *HTTPServer) NotifyLogoutClients(r *http.Request, loginID string) (*LogoutNotificationReport, error) {
	if err := h.initializationError(); err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrInvalidRequest
	}
	if loginID == "" {
		return nil, ErrUserIDEmpty
	}
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	sessions, err := h.server.GetClientSessions(r.Context(), loginID)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(sessions, func(a, b ClientSession) int { return strings.Compare(a.ClientID, b.ClientID) })
	report := &LogoutNotificationReport{Complete: true, Clients: make([]LogoutCallbackResult, len(sessions))}
	jobs := make(chan int)
	var workers sync.WaitGroup
	for i := 0; i < min(len(sessions), h.options.LogoutCallbackConcurrency); i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				report.Clients[index] = h.notifyLogoutTarget(r, sessions[index])
			}
		}()
	}
	for index := range sessions {
		jobs <- index
	}
	close(jobs)
	workers.Wait()

	var failures []error
	for _, result := range report.Clients {
		if result.Cause != nil {
			report.Complete = false
			failures = append(failures, fmt.Errorf("logout callback for client %q: %w", result.ClientID, result.Cause))
		}
	}
	if err := r.Context().Err(); err != nil {
		report.Complete = false
		failures = append(failures, err)
	}
	return report, errors.Join(failures...)
}

// notifyLogoutTarget retries only temporary failures, releasing the shared slot before waiting. notifyLogoutTarget 仅重试暂时性失败，并在等待前释放共享并发槽。
func (h *HTTPServer) notifyLogoutTarget(r *http.Request, session ClientSession) LogoutCallbackResult {
	result := LogoutCallbackResult{ClientID: session.ClientID}
	if session.LogoutCallbackURL == "" {
		result.Status, result.Reason = LogoutCallbackSkipped, "no_callback"
		return result
	}
	ctx := r.Context()
	for result.Attempts < h.options.LogoutCallbackMaxAttempts {
		if err := ctx.Err(); err != nil {
			result.Status, result.Reason, result.Cause = LogoutCallbackCanceled, "canceled", err
			return result
		}
		select {
		case h.logoutSlots <- struct{}{}:
		case <-ctx.Done():
			result.Status, result.Reason, result.Cause = LogoutCallbackCanceled, "canceled", ctx.Err()
			return result
		}
		// Recheck after queueing so a canceled request never starts a fresh delivery. 排队结束后复查，避免已取消请求开始新的通知。
		if err := ctx.Err(); err != nil {
			<-h.logoutSlots
			result.Status, result.Reason, result.Cause = LogoutCallbackCanceled, "canceled", err
			return result
		}
		result.Attempts++
		status, err := h.sendLogoutCallback(r, session)
		<-h.logoutSlots
		result.HTTPStatus, result.Cause = status, err
		if ctx.Err() != nil {
			result.Status, result.Reason, result.Cause = LogoutCallbackCanceled, "canceled", ctx.Err()
			return result
		}
		if errors.Is(err, errLogoutClientRemoved) {
			result.Status, result.Reason, result.Cause = LogoutCallbackSkipped, "unregistered", nil
			return result
		}
		if err == nil {
			result.Status, result.Reason = LogoutCallbackDelivered, ""
			return result
		}
		result.Status = LogoutCallbackFailed
		var failure *logoutDeliveryError
		if !errors.As(err, &failure) {
			result.Reason = "configuration"
			return result
		}
		result.Reason = failure.reason
		if !failure.retryable || result.Attempts == h.options.LogoutCallbackMaxAttempts {
			return result
		}
		timer := time.NewTimer(h.options.LogoutCallbackRetryInterval)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			result.Status, result.Reason, result.Cause = LogoutCallbackCanceled, "canceled", ctx.Err()
			return result
		}
	}
	return result
}

// pushLogoutCallbacks preserves the existing internal strict/best-effort contract. pushLogoutCallbacks 保留现有内部严格与尽力模式契约。
func (h *HTTPServer) pushLogoutCallbacks(r *http.Request, loginID string) error {
	report, err := h.NotifyLogoutClients(r, loginID)
	if err != nil && report != nil && h.options.LogoutCallbackBestEffort && r.Context().Err() == nil {
		return nil
	}
	return err
}

// writeLogoutResult exposes safe diagnostics and invokes the application's synchronous observer. writeLogoutResult 输出安全诊断并调用应用的同步观察钩子。
func (h *HTTPServer) writeLogoutResult(w http.ResponseWriter, r *http.Request, result LogoutResult) {
	if h.options.OnLogoutResult != nil {
		observed := result
		if result.Notifications != nil {
			report := *result.Notifications
			report.Clients = slices.Clone(report.Clients)
			observed.Notifications = &report
		}
		h.options.OnLogoutResult(r, observed)
	}
	if result.Cause != nil {
		status := statusFromError(result.Cause)
		h.observeHTTPError(r, status, result.Cause)
		response := ErrorResponse(status, "logout failed during "+result.Stage)
		response.Data = result
		writeJSON(w, status, response)
		return
	}
	writeJSON(w, http.StatusOK, OKResponse(result))
}

// logoutDeliveryError records retry policy separately from the retained cause. logoutDeliveryError 独立于原始原因记录重试策略。
type logoutDeliveryError struct {
	reason    string
	retryable bool
	cause     error
}

// Error describes the underlying failure for trusted diagnostics. Error 为可信诊断描述底层错误。
func (e *logoutDeliveryError) Error() string { return e.cause.Error() }

// Unwrap preserves errors.Is and errors.As matching. Unwrap 保留 errors.Is 与 errors.As 匹配能力。
func (e *logoutDeliveryError) Unwrap() error { return e.cause }

// deliveryFailure classifies deadline expiration while preserving the original cause. deliveryFailure 分类截止时间超时并保留原始原因。
func deliveryFailure(reason string, retryable bool, err error) error {
	var networkError net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &networkError) && networkError.Timeout()) {
		reason = "timeout"
	}
	return &logoutDeliveryError{reason: reason, retryable: retryable, cause: err}
}
