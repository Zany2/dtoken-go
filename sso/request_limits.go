package sso

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/url"
)

type requestFormErrorKey struct{} // Retain failures when adapters validate the same request again. 接入层重复校验同一请求时保留失败结果。

// ErrRequestTooLarge identifies an inbound SSO query or form exceeding configured limits. ErrRequestTooLarge 标识入站 SSO 查询串或表单超出配置上限。
var ErrRequestTooLarge = errors.New("sso request too large")

// RequestLimits bounds protocol input before signature verification and state consumption. RequestLimits 在验签与状态消费前限制协议输入大小。
type RequestLimits struct {
	// MaxQueryBytes limits the raw query string; nonpositive values use 8 KiB. MaxQueryBytes 限制原始查询串字节数；非正值使用 8 KiB。
	MaxQueryBytes int64
	// MaxBodyBytes limits form body reads; nonpositive values use 64 KiB. MaxBodyBytes 限制表单体读取字节数；非正值使用 64 KiB。
	MaxBodyBytes int64
}

// DefaultRequestLimits returns the inbound query and form limits. DefaultRequestLimits 返回入站查询串与表单限制。
func DefaultRequestLimits() RequestLimits {
	return RequestLimits{MaxQueryBytes: 8 << 10, MaxBodyBytes: 64 << 10}
}

// normalizeRequestLimits applies defaults independently for each boundary. normalizeRequestLimits 分别为每个边界补齐默认值。
func normalizeRequestLimits(limits RequestLimits) RequestLimits {
	defaults := DefaultRequestLimits()
	if limits.MaxQueryBytes <= 0 {
		limits.MaxQueryBytes = defaults.MaxQueryBytes
	}
	if limits.MaxBodyBytes <= 0 {
		limits.MaxBodyBytes = defaults.MaxBodyBytes
	}
	return limits
}

// parseRequestQuery checks wire size before allocating decoded parameters and hides parser diagnostics. parseRequestQuery 在分配解码参数前检查传输大小，并隐藏解析器诊断。
func parseRequestQuery(r *http.Request, limits RequestLimits) (url.Values, error) {
	if r == nil || r.URL == nil {
		return nil, ErrInvalidRequest
	}
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	limits = normalizeRequestLimits(limits)
	if int64(len(r.URL.RawQuery)) > limits.MaxQueryBytes {
		return nil, ErrRequestTooLarge
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, ErrInvalidRequest
	}
	return values, nil
}

// parseRequestForm bounds unread bodies and validates cached middleware forms without re-encoding them. parseRequestForm 限制未读取的请求体，并检查中间件缓存的表单，避免重新编码分配。
func parseRequestForm(r *http.Request, limits RequestLimits) (err error) {
	limits = normalizeRequestLimits(limits)
	if _, err := parseRequestQuery(r, limits); err != nil {
		return err
	}
	if previous, ok := r.Context().Value(requestFormErrorKey{}).(error); ok {
		return previous
	}

	// A consumed body cannot be parsed again to recover from failure; require a fresh request. 已消费请求体不能靠再次解析恢复失败，必须使用新请求。
	defer func() {
		if err != nil {
			*r = *r.WithContext(context.WithValue(r.Context(), requestFormErrorKey{}, err))
		}
	}()

	if r.Method == http.MethodPost {
		if r.ContentLength > limits.MaxBodyBytes {
			return ErrRequestTooLarge
		}
		if r.PostForm == nil && r.Body != nil {
			r.Body = http.MaxBytesReader(nil, r.Body, limits.MaxBodyBytes)
		}
	}

	// Publish parsed forms only after success so partial parser output cannot bypass later checks. 仅在成功后发布解析表单，避免部分解析结果绕过后续检查。
	parsed := *r
	if err := parsed.ParseForm(); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return ErrRequestTooLarge
		}
		return ErrInvalidRequest
	}
	if err := checkFormSize(parsed.PostForm, limits.MaxBodyBytes); err != nil {
		return err
	}
	budget := limits.MaxQueryBytes
	if r.Method == http.MethodPost {
		if limits.MaxBodyBytes > math.MaxInt64-budget {
			budget = math.MaxInt64
		} else {
			budget += limits.MaxBodyBytes
		}
	}
	if err := checkFormSize(parsed.Form, budget); err != nil {
		return err
	}
	r.Form, r.PostForm = parsed.Form, parsed.PostForm
	return nil
}

// checkFormSize bounds decoded keys and values, including repeated keys, without allocating encoded copies. checkFormSize 限制解码后键值（含重复键）的大小，不分配编码副本。
func checkFormSize(values url.Values, budget int64) error {
	for key, items := range values {
		if len(items) == 0 {
			budget -= int64(len(key))
			if budget < 0 {
				return ErrRequestTooLarge
			}
		}
		for _, item := range items {
			budget -= int64(len(key))
			if budget < 0 {
				return ErrRequestTooLarge
			}
			budget -= int64(len(item))
			if budget < 0 {
				return ErrRequestTooLarge
			}
		}
	}
	return nil
}
