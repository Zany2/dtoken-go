#!/usr/bin/env bash
set -euo pipefail

# Accept explicit published versions or use the checkout modules. 接收显式发布版本，默认使用当前源码模块。
if [[ $# != 0 && $# != 2 && $# != 3 ]]; then
  echo 'Usage: bash .github/scripts/check-consumer.sh [root-version gin-version [sso-version]]' >&2
  exit 1
fi
root_version="${1:-v0.0.0}"
gin_version="${2:-v0.0.0}"
sso_version="${3:-v0.0.0}"
check_sso=false
if [[ $# == 0 || $# == 3 ]]; then
  check_sso=true
fi
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
modules=""
if [[ $# == 0 ]]; then
  checkout_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
  cd "$checkout_root"
  checkout_native="$checkout_root"
  if command -v cygpath > /dev/null 2>&1; then
    checkout_native="$(cygpath -m "$checkout_root")"
  fi
  modules="$(GOWORK="$checkout_native/go.work" go list -m -f '{{.Path}}={{.Dir}}')"
elif [[ -z "$1" || -z "$2" || ( $# == 3 && -z "${3-}" ) ]]; then
  echo 'Every requested published version must be provided.' >&2
  exit 1
fi

temp_dir="$(mktemp -d)"
trap 'rm -rf "$temp_dir"' EXIT

cd "$temp_dir"
export GOWORK=off
go mod init example.com/dtoken-consumer
go mod edit "-require=github.com/Zany2/dtoken-go@$root_version"
go mod edit "-require=github.com/Zany2/dtoken-go/integrations/gin@$gin_version"
if [[ "$check_sso" == true ]]; then
  go mod edit "-require=github.com/Zany2/dtoken-go/sso@$sso_version"
  cp "$script_dir/testdata/sso-consumer_test.go" sso_consumer_test.go
fi
while IFS= read -r replacement; do
  if [[ -n "$replacement" ]]; then
    go mod edit -replace="$replacement"
  fi
done <<< "$modules"

cat > consumer_test.go <<'GO'
package consumer_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	dtoken "github.com/Zany2/dtoken-go"
	requestcontext "github.com/Zany2/dtoken-go/httpauth/context"
	"github.com/Zany2/dtoken-go/nonce"
	gindt "github.com/Zany2/dtoken-go/integrations/gin"
	"github.com/gin-gonic/gin"
)

func TestConsumerInstanceRegistryAndCapabilities(t *testing.T) {
	ctx := context.Background()
	var strategy *dtoken.Strategy = dtoken.DefaultStrategy()
	var option dtoken.ManagerOption = dtoken.WithStrategy(strategy)
	var builder *dtoken.Builder = gindt.NewBuilder()
	auth, err := builder.UseManagerOption(option).AuthType("consumer").TokenName("Authorization").AutoRenew(false).IsPrintBanner(false).Build()
	if err != nil {
		t.Fatal(err)
	}
	defer auth.Close()
	token, err := auth.LoginID(ctx, "consumer-user")
	if err != nil {
		t.Fatal(err)
	}
	registered, err := dtoken.GetManager("consumer")
	if err != nil || registered != auth.Manager() {
		t.Fatalf("Build did not register its engine: %v", err)
	}
	if err := dtoken.AuthType("consumer").CheckLogin(ctx, token); err != nil {
		t.Fatal(err)
	}
	if err := dtoken.New(auth.Manager()).CheckLogin(ctx, token); err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	nonceManager, err := nonce.NewDefaultNonceManager(auth.Config().AuthType, auth.Config().KeyPrefix, auth.Storage())
	if err != nil {
		t.Fatal(err)
	}
	nonceService := nonce.NewService(nonceManager, auth)
	authOption, err := gindt.WithAuth(auth)
	if err != nil {
		t.Fatal(err)
	}
	router.Use(gindt.RegisterDTokenContextMiddleware(ctx, authOption, gindt.WithCapabilities(requestcontext.Capabilities{Nonce: nonceService})))
	router.Use(gindt.AuthMiddleware(ctx))
	router.GET("/me", func(c *gin.Context) {
		value, err := gindt.GenerateNonceByContext(c)
		if err != nil {
			t.Error(err)
			c.Status(http.StatusInternalServerError)
			return
		}
		if !nonceService.VerifyNonce(c.Request.Context(), value) {
			t.Error("request used a different nonce service")
		}
		c.String(http.StatusOK, "ok")
	})
	request := httptest.NewRequest(http.MethodGet, "/me", nil)
	request.Header.Set("Authorization", token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "ok" {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
	if err := auth.LogoutByToken(ctx, token); err != nil {
		t.Fatal(err)
	}
	if auth.IsLogin(ctx, token) {
		t.Fatal("token remained active after logout")
	}
	dtoken.CloseAll()
	if !auth.IsClosed() || dtoken.AuthType("consumer").Manager() != nil {
		t.Fatal("CloseAll left the registered engine available")
	}
}
GO

gofmt -w consumer_test.go
go mod tidy

# Verify the selected versions after transitive requirements are resolved. 解析传递依赖后核对最终选中版本。
if [[ $# != 0 ]]; then
  requirements=("github.com/Zany2/dtoken-go=$root_version" "github.com/Zany2/dtoken-go/integrations/gin=$gin_version")
  if [[ "$check_sso" == true ]]; then
    requirements+=("github.com/Zany2/dtoken-go/sso=$sso_version")
  fi
  for requirement in "${requirements[@]}"; do
    module_path="${requirement%=*}"
    expected_version="${requirement#*=}"
    selected_version="$(go list -m -f '{{.Version}}' "$module_path")"
    if [[ "$selected_version" != "$expected_version" ]]; then
      echo "$module_path selected $selected_version, expected $expected_version" >&2
      exit 1
    fi
  done
fi

go vet ./...
go test -count=1 -timeout=5m ./...
