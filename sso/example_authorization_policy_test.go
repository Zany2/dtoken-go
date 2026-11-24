package sso_test

import (
	"context"
	"slices"

	"github.com/Zany2/dtoken-go/sso"
)

// ExampleAuthorizationPolicyFunc demonstrates application-owned membership and scope reduction at every HTTP phase. ExampleAuthorizationPolicyFunc 演示在所有 HTTP 阶段使用应用持有的成员权限并缩减范围。
func ExampleAuthorizationPolicyFunc() {
	// This immutable demo map stands in for an application membership service. 此只读演示映射代表应用成员权限服务。
	permissions := map[string]map[string][]string{
		"orders": {"alice": {"orders:read"}},
	}
	policy := sso.AuthorizationPolicyFunc(func(ctx context.Context, request sso.AuthorizationRequest) (sso.AuthorizationDecision, error) {
		if err := ctx.Err(); err != nil {
			return sso.AuthorizationDecision{}, err
		}
		allowedScopes, member := permissions[request.ClientID][request.LoginID]
		if !member {
			return sso.AuthorizationDecision{}, nil
		}
		granted := make([]string, 0, len(request.Scopes))
		for _, requested := range request.Scopes {
			if slices.Contains(allowedScopes, requested) {
				granted = append(granted, requested)
			}
		}
		return sso.AuthorizationDecision{Allowed: true, Scopes: granted}, nil
	})
	options := sso.DefaultHTTPOptions()
	options.AuthorizationPolicy = policy
	// Configure clients, signing, and LoginIDResolver before passing options to NewHTTPServer. 将 options 传给 NewHTTPServer 前，配置客户端、签名及 LoginIDResolver。
}
