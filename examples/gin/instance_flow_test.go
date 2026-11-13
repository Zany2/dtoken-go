package main

import (
	"bytes"
	"context"
	"encoding/json"
	dtglobal "github.com/Zany2/dtoken-go/global"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zany2/dtoken-go"
	manager "github.com/Zany2/dtoken-go/internal/engine"
	"github.com/gin-gonic/gin"
)

// TestInjectedInstanceOwnsLoginAndRefresh verifies complete flows without global registration. TestInjectedInstanceOwnsLoginAndRefresh 验证无需全局注册的完整实例流程。
func TestInjectedInstanceOwnsLoginAndRefresh(t *testing.T) {
	dtglobal.DeleteAllManager()
	t.Cleanup(dtglobal.DeleteAllManager)
	ctx := context.Background()
	for _, name := range []string{"tenant-a", "tenant-b"} {
		t.Run(name, func(t *testing.T) {
			auth, err := dtoken.NewBuilder().AuthType(name).AutoRenew(false).IsPrintBanner(false).IsLog(false).Build()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(auth.Close)
			var global *dtoken.Auth
			if name == "tenant-b" {
				global, err = dtoken.NewBuilder().AutoRenew(false).IsPrintBanner(false).IsLog(false).Build()
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(global.Close)
				dtglobal.SetManager(global.Manager())
			}
			router := gin.New()
			registerRoutes(router, auth)
			request := func(method, path, body, token string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
				req.Header.Set("Content-Type", "application/json")
				if token != "" {
					req.Header.Set(auth.Manager().GetConfig().TokenName, token)
				}
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				return rec
			}
			decodePair := func(rec *httptest.ResponseRecorder) manager.RefreshTokenPair {
				t.Helper()
				if rec.Code != http.StatusOK {
					t.Fatalf("response = %d %s", rec.Code, rec.Body.String())
				}
				var envelope struct{ Data manager.RefreshTokenPair }
				if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
					t.Fatal(err)
				}
				if envelope.Data.AccessToken == "" || envelope.Data.RefreshToken == "" {
					t.Fatal("response omitted token pair")
				}
				return envelope.Data
			}

			pair := decodePair(request(http.MethodPost, "/login", `{"username":"alice","password":"123456"}`, ""))
			if global != nil && global.IsLogin(ctx, pair.AccessToken) {
				t.Fatal("login used the global manager instead of the injected instance")
			}
			if !auth.Manager().HasRole(ctx, "alice", "admin") || !auth.Manager().HasPermission(ctx, "alice", "article:read") {
				t.Fatal("grants were not stored in the injected manager")
			}
			for _, path := range []string{"/me", "/admin", "/articles"} {
				if rec := request(http.MethodGet, path, "", pair.AccessToken); rec.Code != http.StatusOK {
					t.Fatalf("%s = %d %s", path, rec.Code, rec.Body.String())
				}
			}
			body, err := json.Marshal(RefreshRequest{RefreshToken: pair.RefreshToken})
			if err != nil {
				t.Fatal(err)
			}
			rotated := decodePair(request(http.MethodPost, "/refresh", string(body), ""))
			if auth.IsLogin(ctx, pair.AccessToken) {
				t.Fatal("rotation left the old access token active")
			}
			if rec := request(http.MethodPost, "/logout", "", rotated.AccessToken); rec.Code != http.StatusOK {
				t.Fatalf("logout = %d %s", rec.Code, rec.Body.String())
			}
			if auth.IsLogin(ctx, rotated.AccessToken) {
				t.Fatal("logout left the access token active")
			}
		})
	}
}
