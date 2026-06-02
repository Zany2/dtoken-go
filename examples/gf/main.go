// @Author daixk 2025/12/22 15:56:00
package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	gfdt "github.com/Zany2/dtoken-go/integrations/gf"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

func main() {
	ctx := context.Background()
	initDToken()

	s := g.Server()
	s.Use(gfdt.RegisterDTokenContextMiddleware(ctx))
	s.Group("/", func(group *ghttp.RouterGroup) {
		group.POST("/login", handleLogin)
		group.POST("/refresh", handleRefresh)
	})

	s.Group("/", func(group *ghttp.RouterGroup) {
		group.Middleware(gfdt.AuthMiddleware(ctx))
		group.GET("/me", handleMe)
		group.GET("/introspect", handleIntrospect)
		group.GET("/admin", gfdt.CheckRoleMiddleware(ctx, []string{"admin"}, handleAdmin, nil))
		group.GET("/articles", gfdt.CheckPermissionMiddleware(ctx, []string{"article:read"}, handleArticles, nil))
		group.POST("/logout", handleLogout)
	})

	s.Group("/access", func(group *ghttp.RouterGroup) {
		group.Middleware(gfdt.AccessMiddleware(ctx,
			gfdt.WithRouteAccessHandler(resolveRouteAccess),
			gfdt.WithFailFunc(handleAuthFail),
		))
		group.GET("/public", handleAccessPublic)
		group.GET("/me", handleMe)
		group.GET("/articles", handleArticles)
		group.GET("/admin", handleAdmin)
	})

	s.SetPort(8080)
	s.Run()
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() {
	mgr, err := gfdt.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
		RefreshTokenTimeout(int64((30 * 24 * time.Hour).Seconds())).
		IsPrintBanner(false).
		Build()
	if err != nil {
		panic(err)
	}

	gfdt.SetManager(mgr)
}

// handleLogin logs in a demo user handleLogin 登录示例用户
func handleLogin(r *ghttp.Request) {
	username := r.Get("username").String()
	password := r.Get("password").String()
	if username == "" || password == "" {
		writeJSON(r, http.StatusBadRequest, gfdt.CodeBadRequest, "username and password are required", nil)
		return
	}

	if password != "123456" {
		writeJSON(r, http.StatusUnauthorized, gfdt.CodeNotLogin, "invalid username or password", nil)
		return
	}

	pair, err := gfdt.LoginWithRefreshToken(r.Context(), username, "web", "gf-example")
	if err != nil {
		writeJSON(r, http.StatusInternalServerError, gfdt.CodeServerError, err.Error(), nil)
		return
	}

	// Seed demo authorization data 初始化示例权限数据
	_ = gfdt.AddRoles(r.Context(), username, []string{"admin"})
	_ = gfdt.AddPermissions(r.Context(), username, []string{"article:read"})

	writeJSON(r, http.StatusOK, gfdt.CodeSuccess, "ok", pair)
}

// handleRefresh rotates refresh token handleRefresh 轮换刷新令牌
func handleRefresh(r *ghttp.Request) {
	refreshToken := r.Get("refreshToken").String()
	if refreshToken == "" {
		writeJSON(r, http.StatusBadRequest, gfdt.CodeBadRequest, "refreshToken is required", nil)
		return
	}

	pair, err := gfdt.RefreshToken(r.Context(), refreshToken)
	if err != nil {
		writeJSON(r, http.StatusUnauthorized, gfdt.CodeNotLogin, err.Error(), nil)
		return
	}

	writeJSON(r, http.StatusOK, gfdt.CodeSuccess, "ok", pair)
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(r *ghttp.Request) {
	dCtx, ok := gfdt.GetDTokenContext(r)
	if !ok {
		writeJSON(r, http.StatusUnauthorized, gfdt.CodeNotLogin, "not logged in", nil)
		return
	}

	loginID, err := dCtx.GetLoginID(r.Context())
	if err != nil {
		writeJSON(r, http.StatusUnauthorized, gfdt.CodeNotLogin, err.Error(), nil)
		return
	}

	roles, _ := dCtx.GetRoles(r.Context())
	permissions, _ := dCtx.GetPermissions(r.Context())

	writeJSON(r, http.StatusOK, gfdt.CodeSuccess, "ok", g.Map{
		"loginId":     loginID,
		"roles":       roles,
		"permissions": permissions,
	})
}

// handleIntrospect returns current token introspection handleIntrospect 返回当前 token 自省结果
func handleIntrospect(r *ghttp.Request) {
	info, err := gfdt.IntrospectTokenByCtx(r.Context())
	if err != nil {
		writeJSON(r, http.StatusUnauthorized, gfdt.CodeNotLogin, err.Error(), nil)
		return
	}

	writeJSON(r, http.StatusOK, gfdt.CodeSuccess, "ok", info)
}

// handleAdmin returns admin data handleAdmin 返回管理员数据
func handleAdmin(r *ghttp.Request) {
	writeJSON(r, http.StatusOK, gfdt.CodeSuccess, "ok", g.Map{"scope": "admin"})
}

// handleArticles returns protected article data handleArticles 返回受保护的文章数据
func handleArticles(r *ghttp.Request) {
	writeJSON(r, http.StatusOK, gfdt.CodeSuccess, "ok", []string{"article-a", "article-b"})
}

// handleLogout logs out current token handleLogout 注销当前 Token
func handleLogout(r *ghttp.Request) {
	dCtx, ok := gfdt.GetDTokenContext(r)
	if !ok {
		writeJSON(r, http.StatusUnauthorized, gfdt.CodeNotLogin, "not logged in", nil)
		return
	}

	if err := dCtx.Logout(r.Context()); err != nil {
		writeJSON(r, http.StatusInternalServerError, gfdt.CodeServerError, err.Error(), nil)
		return
	}

	writeJSON(r, http.StatusOK, gfdt.CodeSuccess, "ok", nil)
}

// resolveRouteAccess maps URL paths to auth, permission, and role rules resolveRouteAccess 将 URL 映射为认证、权限、角色规则
func resolveRouteAccess(_ context.Context, r *ghttp.Request, req *gfdt.RouteAccessRequest) {
	path := strings.TrimRight(r.URL.Path, "/")
	switch path {
	case "/access/public":
		req.SkipAuth()
	case "/access/me":
		req.SkipPermission()
	case "/access/articles":
		req.RequirePermissions("article:read")
	case "/access/admin":
		req.RequireRoles("admin")
	}
}

// handleAccessPublic returns public access data handleAccessPublic 返回公开访问数据
func handleAccessPublic(r *ghttp.Request) {
	writeJSON(r, http.StatusOK, gfdt.CodeSuccess, "ok", g.Map{"scope": "public"})
}

// handleAuthFail writes custom auth failure response handleAuthFail 写入自定义认证失败响应
func handleAuthFail(r *ghttp.Request, err error) {
	code := gfdt.CodeNotLogin
	status := http.StatusUnauthorized
	if errors.Is(err, gfdt.ErrPermissionDenied) || errors.Is(err, gfdt.ErrRoleDenied) {
		code = gfdt.CodePermissionDenied
		status = http.StatusForbidden
	}
	writeJSON(r, status, code, err.Error(), nil)
}

// writeJSON writes a unified JSON response writeJSON 写入统一 JSON 响应
func writeJSON(r *ghttp.Request, httpStatus int, code int, message string, data interface{}) {
	r.Response.WriteStatus(httpStatus)
	r.Response.WriteJson(g.Map{
		"code":    code,
		"message": message,
		"data":    data,
	})
}
