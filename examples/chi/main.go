// @Author daixk 2025/12/22 15:56:00
package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	chidt "github.com/Zany2/dtoken-go/integrations/chi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Response defines the example response body Response 定义示例响应结构
type Response struct {
	// Code stores the application response code. Code 保存应用响应码。
	Code int `json:"code"`
	// Message stores the response message. Message 保存响应消息。
	Message string `json:"message"`
	// Data stores the optional response payload. Data 保存可选响应数据。
	Data interface{} `json:"data,omitempty"`
}

// LoginRequest defines the login payload LoginRequest 定义登录请求参数
type LoginRequest struct {
	// Username stores the demo login name. Username 保存示例登录名。
	Username string `json:"username"`
	// Password stores the demo login password. Password 保存示例登录密码。
	Password string `json:"password"`
}

func main() {
	initDToken()
	defer chidt.DeleteAllManager()

	r := chi.NewRouter()
	registerRoutes(r)

	if err := http.ListenAndServe(":8080", r); err != nil {
		panic(err)
	}
}

// registerRoutes shares production middleware and routes with regression tests. registerRoutes 与回归测试共用实际中间件和路由。
func registerRoutes(r chi.Router) {
	fail := func(w http.ResponseWriter, _ *http.Request, err error) { writeAuthError(w, err) }
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(chidt.RegisterDTokenContextMiddleware(chidt.WithFailFunc(fail)))
	r.Post("/login", handleLogin)

	r.Group(func(auth chi.Router) {
		auth.Use(chidt.AuthMiddleware(chidt.WithFailFunc(fail)))
		auth.Get("/me", handleMe)
		auth.With(chidt.RoleMiddleware([]string{"admin"}, chidt.WithFailFunc(fail))).Get("/admin", handleAdmin)
		auth.With(chidt.PermissionMiddleware([]string{"article:read"}, chidt.WithFailFunc(fail))).Get("/articles", handleArticles)
		auth.Post("/logout", handleLogout)
	})
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() {
	mgr, err := chidt.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
		RenewMaxRefresh(int64(time.Hour.Seconds())).
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}

	chidt.SetManager(mgr)
}

// handleLogin logs in a demo user handleLogin 登录示例用户
func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, chidt.CodeBadRequest, "username and password are required", nil)
		return
	}

	// Require a single JSON value before login can change state. 登录改变状态前，确认请求体只包含一个 JSON 值。
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, chidt.CodeBadRequest, "request body must contain a single JSON object", nil)
		return
	}

	if req.Password != "123456" {
		writeJSON(w, http.StatusUnauthorized, chidt.CodeNotLogin, "invalid username or password", nil)
		return
	}

	token, err := chidt.Login(r.Context(), req.Username)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	// Grant every demo user the same role and permission; real apps must load their own access rules. 为每个演示用户授予相同角色和权限；实际应用应加载自身的授权规则。
	if err = chidt.AddRoles(r.Context(), req.Username, []string{"admin"}); err != nil {
		writeAuthError(w, err)
		return
	}
	if err = chidt.AddPermissions(r.Context(), req.Username, []string{"article:read"}); err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, chidt.CodeSuccess, "ok", map[string]interface{}{"token": token})
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(w http.ResponseWriter, r *http.Request) {
	dCtx, ok := chidt.GetDTokenContextByCtx(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, chidt.CodeNotLogin, "not logged in", nil)
		return
	}

	loginID, err := dCtx.Auth().GetLoginID(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}

	roles, err := dCtx.Access().GetRoles(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	permissions, err := dCtx.Access().GetPermissions(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, chidt.CodeSuccess, "ok", map[string]interface{}{
		"loginId":     loginID,
		"roles":       roles,
		"permissions": permissions,
	})
}

// handleAdmin returns admin data handleAdmin 返回管理员数据
func handleAdmin(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, chidt.CodeSuccess, "ok", map[string]interface{}{"scope": "admin"})
}

// handleArticles returns protected article data handleArticles 返回受保护的文章数据
func handleArticles(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, chidt.CodeSuccess, "ok", []string{"article-a", "article-b"})
}

// handleLogout logs out current token handleLogout 注销当前 Token
func handleLogout(w http.ResponseWriter, r *http.Request) {
	dCtx, ok := chidt.GetDTokenContextByCtx(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, chidt.CodeNotLogin, "not logged in", nil)
		return
	}

	if err := dCtx.Auth().Logout(r.Context()); err != nil {
		writeAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, chidt.CodeSuccess, "ok", nil)
}

// writeAuthError distinguishes credential failures from access restrictions and server failures. writeAuthError 区分凭证无效、访问限制及服务端故障。
func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, chidt.ErrNotLogin), errors.Is(err, chidt.ErrInvalidToken),
		errors.Is(err, chidt.ErrTokenExpired), errors.Is(err, chidt.ErrActiveTimeout),
		errors.Is(err, chidt.ErrTokenKickout), errors.Is(err, chidt.ErrTokenReplaced):
		writeJSON(w, http.StatusUnauthorized, chidt.CodeNotLogin, err.Error(), nil)
	case errors.Is(err, chidt.ErrAccountDisabled), errors.Is(err, chidt.ErrDeviceDisabled):
		writeJSON(w, http.StatusForbidden, chidt.CodeAccountDisabled, err.Error(), nil)
	case errors.Is(err, chidt.ErrPermissionDenied), errors.Is(err, chidt.ErrRoleDenied):
		writeJSON(w, http.StatusForbidden, chidt.CodePermissionDenied, err.Error(), nil)
	default:
		// Keep server details in logs rather than HTTP responses. 将服务端详情保留在日志中，不在 HTTP 响应中暴露。
		log.Printf("dtoken request failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, chidt.CodeServerError, "internal server error", nil)
	}
}

// writeJSON writes a unified JSON response writeJSON 写入统一 JSON 响应
func writeJSON(w http.ResponseWriter, httpStatus int, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
