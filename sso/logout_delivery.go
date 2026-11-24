package sso

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var errLogoutClientRemoved = errors.New("logout callback client is no longer registered") // Internal skip marker. 内部跳过标记。

// postLogoutCallback preserves direct delivery semantics for existing internal callers. postLogoutCallback 为现有内部调用方保留直接投递语义。
func (h *HTTPServer) postLogoutCallback(r *http.Request, session ClientSession) error {
	_, err := h.sendLogoutCallback(r, session)
	if errors.Is(err, errLogoutClientRemoved) {
		return nil
	}
	return err
}

// sendLogoutCallback sends one freshly signed attempt, classifying failures for bounded retries. sendLogoutCallback 发送一次新签名通知，并分类错误以支持有限重试。
func (h *HTTPServer) sendLogoutCallback(r *http.Request, session ClientSession) (int, error) {
	ctx := r.Context()
	if h.options.LogoutCallbackTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.options.LogoutCallbackTimeout)
		defer cancel()
	}
	if err := ctx.Err(); err != nil {
		return 0, deliveryFailure("transport", true, err)
	}

	// Check current registration and allow-lists on every attempt, including retries. 每次尝试（包括重试）都检查当前登记与回调白名单。
	registered, err := h.server.getClient(ctx, session.ClientID)
	if contextErr := ctx.Err(); contextErr != nil {
		return 0, deliveryFailure("storage", true, errors.Join(err, contextErr))
	}
	if errors.Is(err, ErrClientNotFound) {
		return 0, errLogoutClientRemoved
	}
	if err != nil {
		return 0, deliveryFailure("storage", true, err)
	}
	if !h.server.isValidLogoutCallbackURL(registered, session.LogoutCallbackURL) {
		return 0, deliveryFailure("configuration", false, ErrInvalidCallbackURL)
	}

	values := url.Values{}
	values.Set(h.options.Params.LoginID, session.LoginID)
	values.Set(h.options.Params.Client, session.ClientID)
	values.Set(h.options.Params.Timestamp, time.Now().Format(time.RFC3339))
	if err := h.options.ReplayProtection.validateConfig(h.options.CheckSign, h.options.Params); err != nil {
		return 0, deliveryFailure("configuration", false, err)
	}
	if h.options.ReplayProtection != nil {
		values = h.options.ReplayProtection.stamp(values, h.options.Params, ActionLogoutCallback)
	}
	if h.options.CheckSign {
		secret, err := h.signingSecret(registered)
		if err != nil {
			return 0, deliveryFailure("configuration", false, err)
		}
		values = NewSignerWithParams(secret, h.options.Params).AttachSign(values)
	}
	if err := ctx.Err(); err != nil {
		return 0, deliveryFailure("transport", true, err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, session.LogoutCallbackURL, strings.NewReader(values.Encode()))
	if err != nil {
		return 0, deliveryFailure("configuration", false, err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := h.options.LogoutHTTPClient
	if client == nil {
		client = &http.Client{}
	}
	if client.CheckRedirect == nil {
		// Preserve caller transport settings without silently forwarding identity to a redirect. 保留调用方传输配置，不隐式将身份转发到重定向目标。
		clientCopy := *client
		clientCopy.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
		client = &clientCopy
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, deliveryFailure("transport", true, err)
	}
	defer response.Body.Close()
	status := response.StatusCode
	if err := ctx.Err(); err != nil {
		return status, deliveryFailure("transport", true, err)
	}
	if status < 200 || status >= 300 {
		retryable := status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || (status >= 500 && status <= 599)
		return status, deliveryFailure("http_status", retryable, fmt.Errorf("logout callback failed with status %d", status))
	}

	// Bound and validate acknowledgements; a 2xx status alone does not imply business success. 限制并校验确认响应，2xx 状态本身不代表业务成功。
	const maxAcknowledgementSize = 64 << 10
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxAcknowledgementSize+1))
	if err != nil {
		return status, deliveryFailure("transport", true, fmt.Errorf("read logout callback response: %w", err))
	}

	// Enforce the attempt deadline even when a custom body reader returns success after cancellation. 即使自定义响应体读取器取消后仍返回成功，也执行本次尝试的截止时间。
	if err := ctx.Err(); err != nil {
		return status, deliveryFailure("transport", true, err)
	}
	if len(payload) > maxAcknowledgementSize {
		return status, deliveryFailure("acknowledgement", false, errors.New("logout callback response exceeds 64 KiB"))
	}
	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 || string(payload) == ResultOK {
		return status, nil
	}
	var acknowledgement struct {
		Code *int `json:"code"`
	}
	if err := json.Unmarshal(payload, &acknowledgement); err != nil {
		return status, deliveryFailure("acknowledgement", false, fmt.Errorf("invalid logout callback response: %w", err))
	}
	if acknowledgement.Code == nil {
		return status, deliveryFailure("acknowledgement", false, errors.New("logout callback response is missing code"))
	}
	if *acknowledgement.Code != 0 {
		return status, deliveryFailure("acknowledgement", false, fmt.Errorf("logout callback failed with code %d", *acknowledgement.Code))
	}
	return status, nil
}
