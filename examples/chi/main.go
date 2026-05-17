// @Author daixk 2025/12/22 15:56:00
package main

import (
	"encoding/json"
	"net/http"
	"time"

	chidt "github.com/Zany2/dtoken-go/integrations/chi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Response defines the example response body Response 定义示例响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// LoginRequest defines the login payload LoginRequest 定义登录请求参数
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func main() {
	initDToken()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(chidt.RegisterDTokenContextMiddleware())
	r.Post("/login", handleLogin)

	r.Group(func(auth chi.Router) {
		auth.Use(chidt.AuthMiddleware())
		auth.Get("/me", handleMe)
		auth.With(chidt.RoleMiddleware([]string{"admin"})).Get("/admin", handleAdmin)
		auth.With(chidt.PermissionMiddleware([]string{"article:read"})).Get("/articles", handleArticles)
		auth.Post("/logout", handleLogout)
	})

	_ = http.ListenAndServe(":8080", r)
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() {
	mgr, err := chidt.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, chidt.CodeBadRequest, "username and password are required", nil)
		return
	}

	if req.Password != "123456" {
		writeJSON(w, http.StatusUnauthorized, chidt.CodeNotLogin, "invalid username or password", nil)
		return
	}

	token, err := chidt.Login(r.Context(), req.Username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, chidt.CodeServerError, err.Error(), nil)
		return
	}

	// Seed demo authorization data 初始化示例权限数据
	_ = chidt.AddRoles(r.Context(), req.Username, []string{"admin"})
	_ = chidt.AddPermissions(r.Context(), req.Username, []string{"article:read"})

	writeJSON(w, http.StatusOK, chidt.CodeSuccess, "ok", map[string]interface{}{"token": token})
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(w http.ResponseWriter, r *http.Request) {
	dCtx, ok := chidt.GetDTokenContextByCtx(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, chidt.CodeNotLogin, "not logged in", nil)
		return
	}

	loginID, err := dCtx.GetLoginID(r.Context())
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, chidt.CodeNotLogin, err.Error(), nil)
		return
	}

	roles, _ := dCtx.GetRoles(r.Context())
	permissions, _ := dCtx.GetPermissions(r.Context())

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

	if err := dCtx.Logout(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, chidt.CodeServerError, err.Error(), nil)
		return
	}

	writeJSON(w, http.StatusOK, chidt.CodeSuccess, "ok", nil)
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
