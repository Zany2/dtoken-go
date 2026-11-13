// @Author daixk 2025/12/22 15:56:00
package main

import (
	dterror "github.com/Zany2/dtoken-go/errors"
	dt1 "github.com/Zany2/dtoken-go/global"

	"context"
	"encoding/json"
	"errors"
	"github.com/Zany2/dtoken-go"
	"io"
	"mime"
	"net/http"
	"time"

	gfdt "github.com/Zany2/dtoken-go/integrations/gf"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

func main() {
	initDToken()
	defer dt1.DeleteAllManager()

	s := g.Server()
	registerRoutes(s)
	s.SetPort(8080)
	if err := s.Start(); err != nil {
		panic(err)
	}
	ghttp.Wait()
}

// registerRoutes shares the example's middleware and routes with regression tests. registerRoutes 与回归测试共用示例中间件和路由。
func registerRoutes(s *ghttp.Server) {
	ctx := context.Background()
	s.Use(gfdt.RegisterDTokenContextMiddleware(ctx, gfdt.WithManagerResolver(dt1.GetManager), gfdt.WithFailFunc(handleAuthFail)))
	s.Group("/", func(group *ghttp.RouterGroup) {
		group.POST("/login", handleLogin)
		group.POST("/refresh", handleRefresh)
	})

	s.Group("/", func(group *ghttp.RouterGroup) {
		group.Middleware(gfdt.AuthMiddleware(ctx, gfdt.WithFailFunc(handleAuthFail)))
		group.GET("/me", handleMe)
		group.GET("/introspect", handleIntrospect)
		group.GET("/admin", gfdt.CheckRoleMiddleware(ctx, []string{"admin"}, handleAdmin, handleAuthFail))
		group.GET("/articles", gfdt.CheckPermissionMiddleware(ctx, []string{"article:read"}, handleArticles, handleAuthFail))
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
}

// initDToken initializes integration manager initDToken 初始化集成管理器
func initDToken() {
	builtAuth1, err := dtoken.NewBuilder().
		Timeout(int64((2 * time.Hour).Seconds())).
		RenewMaxRefresh(int64(time.Hour.Seconds())).
		RefreshTokenTimeout(int64((30 * 24 * time.Hour).Seconds())).
		IsPrintBanner(false).
		Build()
	mgr := builtAuth1.Manager()

	if err != nil {
		panic(err)
	}

	dt1.SetManager(mgr)
}

// handleLogin logs in a demo user handleLogin 登录示例用户
func handleLogin(r *ghttp.Request) {
	values, err := readBodyValues(r)
	username, password := values["username"], values["password"]
	if err != nil || username == "" || password == "" {
		writeJSON(r, http.StatusBadRequest, dterror.CodeBadRequest, "username and password are required", nil)
		return
	}

	if password != "123456" {
		writeJSON(r, http.StatusUnauthorized, dterror.CodeNotLogin, "invalid username or password", nil)
		return
	}

	pair, err := dt1.LoginWithRefreshToken(r.Context(), username, "web", "gf-example")
	if err != nil {
		handleAuthFail(r, err)
		return
	}

	// Grant every demo user the same role and permission; real apps must load their own access rules. 为每个演示用户授予相同角色和权限；实际应用应加载自身的授权规则。
	if err = dt1.AddRoles(r.Context(), username, []string{"admin"}); err != nil {
		handleAuthFail(r, err)
		return
	}
	if err = dt1.AddPermissions(r.Context(), username, []string{"article:read"}); err != nil {
		handleAuthFail(r, err)
		return
	}

	writeJSON(r, http.StatusOK, dterror.CodeSuccess, "ok", pair)
}

// handleRefresh rotates refresh token handleRefresh 轮换刷新令牌
func handleRefresh(r *ghttp.Request) {
	values, err := readBodyValues(r)
	refreshToken := values["refreshToken"]
	if err != nil || refreshToken == "" {
		writeJSON(r, http.StatusBadRequest, dterror.CodeBadRequest, "refreshToken is required", nil)
		return
	}

	pair, err := dt1.RefreshToken(r.Context(), refreshToken)
	if err != nil {
		handleAuthFail(r, err)
		return
	}

	writeJSON(r, http.StatusOK, dterror.CodeSuccess, "ok", pair)
}

// handleMe returns current login information handleMe 返回当前登录信息
func handleMe(r *ghttp.Request) {
	dCtx, ok := gfdt.GetDTokenContext(r)
	if !ok {
		writeJSON(r, http.StatusUnauthorized, dterror.CodeNotLogin, "not logged in", nil)
		return
	}

	loginID, err := dCtx.Auth().GetLoginID(r.Context())
	if err != nil {
		handleAuthFail(r, err)
		return
	}

	roles, err := dCtx.Access().GetRoles(r.Context())
	if err != nil {
		handleAuthFail(r, err)
		return
	}
	permissions, err := dCtx.Access().GetPermissions(r.Context())
	if err != nil {
		handleAuthFail(r, err)
		return
	}

	writeJSON(r, http.StatusOK, dterror.CodeSuccess, "ok", g.Map{
		"loginId":     loginID,
		"roles":       roles,
		"permissions": permissions,
	})
}

// handleIntrospect returns current token introspection handleIntrospect 返回当前 token 自省结果
func handleIntrospect(r *ghttp.Request) {
	info, err := gfdt.IntrospectTokenByCtx(r.Context())
	if err != nil {
		handleAuthFail(r, err)
		return
	}

	writeJSON(r, http.StatusOK, dterror.CodeSuccess, "ok", info)
}

// handleAdmin returns admin data handleAdmin 返回管理员数据
func handleAdmin(r *ghttp.Request) {
	writeJSON(r, http.StatusOK, dterror.CodeSuccess, "ok", g.Map{"scope": "admin"})
}

// handleArticles returns protected article data handleArticles 返回受保护的文章数据
func handleArticles(r *ghttp.Request) {
	writeJSON(r, http.StatusOK, dterror.CodeSuccess, "ok", []string{"article-a", "article-b"})
}

// handleLogout logs out current token handleLogout 注销当前 Token
func handleLogout(r *ghttp.Request) {
	dCtx, ok := gfdt.GetDTokenContext(r)
	if !ok {
		writeJSON(r, http.StatusUnauthorized, dterror.CodeNotLogin, "not logged in", nil)
		return
	}

	if err := dCtx.Auth().Logout(r.Context()); err != nil {
		handleAuthFail(r, err)
		return
	}

	writeJSON(r, http.StatusOK, dterror.CodeSuccess, "ok", nil)
}

// resolveRouteAccess uses the matched route so path normalization and overrides cannot skip checks. resolveRouteAccess 根据实际匹配路由鉴权，避免路径规范化或覆盖导致漏检。
func resolveRouteAccess(_ context.Context, r *ghttp.Request, req *gfdt.RouteAccessRequest) {
	handler := r.GetServeHandler()
	if handler == nil {
		return
	}
	switch handler.Handler.Router.Uri {
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
	writeJSON(r, http.StatusOK, dterror.CodeSuccess, "ok", g.Map{"scope": "public"})
}

// handleAuthFail distinguishes credential failures, access restrictions and server failures. handleAuthFail 区分凭证无效、访问限制及服务端故障。
func handleAuthFail(r *ghttp.Request, err error) {
	switch {
	case errors.Is(err, dterror.ErrNotLogin), errors.Is(err, dterror.ErrInvalidToken),
		errors.Is(err, dterror.ErrInvalidRefreshToken), errors.Is(err, dterror.ErrInvalidAccessToken),
		errors.Is(err, dterror.ErrTokenExpired), errors.Is(err, dterror.ErrActiveTimeout),
		errors.Is(err, dterror.ErrTokenKickout), errors.Is(err, dterror.ErrTokenReplaced):
		writeJSON(r, http.StatusUnauthorized, dterror.CodeNotLogin, err.Error(), nil)
	case errors.Is(err, dterror.ErrAccountDisabled), errors.Is(err, dterror.ErrDeviceDisabled):
		writeJSON(r, http.StatusForbidden, dterror.CodeAccountDisabled, err.Error(), nil)
	case errors.Is(err, dterror.ErrPermissionDenied), errors.Is(err, dterror.ErrRoleDenied):
		writeJSON(r, http.StatusForbidden, dterror.CodePermissionDenied, err.Error(), nil)
	default:
		// Keep server details in logs rather than HTTP responses. 将服务端详情保留在日志中，不在 HTTP 响应中暴露。
		g.Log().Error(r.Context(), "dtoken request failed:", err)
		writeJSON(r, http.StatusInternalServerError, dterror.CodeServerError, "internal server error", nil)
	}
}

// readBodyValues accepts string credentials from JSON or forms without query fallback or implicit conversion. readBodyValues 从 JSON 或表单读取字符串凭证，不回退到 Query 或隐式转换类型。
func readBodyValues(r *ghttp.Request) (map[string]string, error) {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	values := make(map[string]string)
	switch contentType {
	case "application/json":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(body, &values); err != nil {
			return nil, err
		}
	case "application/x-www-form-urlencoded", "multipart/form-data":
		if contentType == "multipart/form-data" {
			err = r.ParseMultipartForm(32 << 20)
			if r.MultipartForm != nil {
				defer r.MultipartForm.RemoveAll()
			}
		} else {
			err = r.Request.ParseForm()
		}
		if err != nil {
			return nil, err
		}
		for key := range r.PostForm {
			values[key] = r.PostForm.Get(key)
		}
	default:
		return nil, errors.New("expected JSON or form body")
	}
	return values, nil
}

// writeJSON writes a unified JSON response writeJSON 写入统一 JSON 响应
func writeJSON(r *ghttp.Request, httpStatus int, code int, message string, data interface{}) {
	r.Response.WriteHeader(httpStatus)
	r.Response.WriteJson(g.Map{
		"code":    code,
		"message": message,
		"data":    data,
	})
}
