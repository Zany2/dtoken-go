#!/usr/bin/env bash
set -euo pipefail

# Check SSO releases independently of Gin and workspace toolchain discovery. 独立于 Gin 及工作区工具链发现检查 SSO 发布模块。
if [[ $# != 0 && $# != 4 ]]; then
  echo 'Usage: bash .github/scripts/check-sso-consumer.sh [root-version sso-version redis-version sso-redis-version]' >&2
  exit 1
fi
for version in "$@"; do
  if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.+-]+)?$ ]]; then
    echo 'Provide four explicit module versions, not branches or latest.' >&2
    exit 1
  fi
done
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
checkout_root="$(cd "$script_dir/../.." && pwd)"
checkout_native="$checkout_root"
if command -v cygpath > /dev/null 2>&1; then
  checkout_native="$(cygpath -m "$checkout_root")"
fi
module_paths=(github.com/Zany2/dtoken-go github.com/Zany2/dtoken-go/sso github.com/Zany2/dtoken-go/storage/redis github.com/Zany2/dtoken-go/sso/storage/redis)
relative_paths=(. sso storage/redis sso/storage/redis)
versions=("${1:-v0.0.0}" "${2:-v0.0.0}" "${3:-v0.0.0}" "${4:-v0.0.0}")
temp_dir="$(mktemp -d)"
trap 'rm -rf "$temp_dir"' EXIT
cd "$temp_dir"
export GOWORK=off
export GOTOOLCHAIN=local
go version
go mod init example.com/dtoken-sso-consumer
for index in "${!module_paths[@]}"; do
  go mod edit "-require=${module_paths[$index]}@${versions[$index]}"
  if [[ $# == 0 ]]; then
    go mod edit "-replace=${module_paths[$index]}=$checkout_native/${relative_paths[$index]}"
  fi
done
cp "$script_dir/testdata/sso-consumer_test.go" sso_consumer_test.go
cp "$script_dir/testdata/sso-browser-consumer_test.go" sso_browser_consumer_test.go
cp "$script_dir/testdata/sso-redis-consumer_test.go" sso_redis_consumer_test.go
go mod tidy

# Refuse version promotion or any published-mode replacement after dependency resolution. 解析依赖后拒绝发布模式下版本提升或任何模块替换。
if [[ $# == 4 ]]; then
  replacements="$(go list -m -f '{{if .Replace}}{{.Path}}{{end}}' all)"
  if [[ -n "$replacements" ]]; then
    echo 'Published consumer unexpectedly resolved replacement modules.' >&2
    exit 1
  fi
  for index in "${!module_paths[@]}"; do
    selected="$(go list -m -f '{{.Version}}' "${module_paths[$index]}")"
    if [[ "$selected" != "${versions[$index]}" ]]; then
      echo "${module_paths[$index]} selected $selected, expected ${versions[$index]}" >&2
      exit 1
    fi
  done
fi
go list -m "${module_paths[@]}"
go vet -mod=readonly ./...
go test -mod=readonly -race -count=1 -timeout=5m ./...
