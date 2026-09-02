package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Zany2/dtoken-go/dtoken"
	gfdt "github.com/Zany2/dtoken-go/integrations/gf"
	"github.com/gogf/gf/v2/net/ghttp"
)

func TestResolveRouteAccessRules(t *testing.T) {
	tests := []struct {
		path           string
		wantPermission string
		wantRole       string
	}{
		{path: "/access/public"},
		{path: "/access/public/"},
		{path: "/access/me"},
		{path: "/access/articles", wantPermission: "article:read"},
		{path: "/access/admin", wantRole: "admin"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			r := &ghttp.Request{Request: httptest.NewRequest(http.MethodGet, tt.path, nil)}
			req := &gfdt.RouteAccessRequest{}
			resolveRouteAccess(context.Background(), r, req)
			if tt.wantPermission != "" && (len(req.Permissions) != 1 || req.Permissions[0] != tt.wantPermission) {
				t.Fatalf("permissions = %v, want [%s]", req.Permissions, tt.wantPermission)
			}
			if tt.wantRole != "" && (len(req.Roles) != 1 || req.Roles[0] != tt.wantRole) {
				t.Fatalf("roles = %v, want [%s]", req.Roles, tt.wantRole)
			}
		})
	}
}

func TestInitDTokenRegistersManager(t *testing.T) {
	dtoken.DeleteAllManager()
	t.Cleanup(dtoken.DeleteAllManager)
	initDToken()

	mgr, err := dtoken.GetManager()
	if err != nil {
		t.Fatalf("GetManager() error = %v", err)
	}
	if mgr.GetConfig().Timeout != 2*60*60 {
		t.Fatalf("timeout = %d, want 7200", mgr.GetConfig().Timeout)
	}
	if mgr.GetConfig().RefreshTokenTimeout <= 0 {
		t.Fatal("refresh token timeout should be configured")
	}
}
