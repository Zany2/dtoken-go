#!/usr/bin/env bash
set -euo pipefail

# Resolve the checkout modules before disabling workspace resolution. 关闭工作区解析前获取当前源码模块。
checkout_root="$(pwd)"
modules="$(go list -m -f '{{.Path}}={{.Dir}}')"
temp_dir="$(mktemp -d)"
trap 'rm -rf "$temp_dir"' EXIT

while IFS='=' read -r module_path module_dir; do
  echo "::group::Standalone - $module_path"
  (
    cd "$module_dir"
    export GOWORK=off

    if [[ "$module_dir" == "$checkout_root/com/codec/jsonv2" ]]; then
      export GOEXPERIMENT=jsonv2
    fi

    # Keep dependency resolution changes in a temporary module file. 将依赖解析产生的变更限制在临时模块文件中。
    cp go.mod "$temp_dir/check.go.mod"
    : > "$temp_dir/check.go.sum"
    if [[ -f go.sum ]]; then
      cat go.sum >> "$temp_dir/check.go.sum"
    fi
    cat "$checkout_root/go.sum" >> "$temp_dir/check.go.sum"
    sort -u "$temp_dir/check.go.sum" -o "$temp_dir/check.go.sum"

    while IFS= read -r replacement; do
      go mod edit -modfile="$temp_dir/check.go.mod" -replace="$replacement"
    done <<< "$modules"

    go vet -mod=readonly -modfile="$temp_dir/check.go.mod" ./...
    go test -mod=readonly -modfile="$temp_dir/check.go.mod" -count=1 -timeout=5m ./...
  )
  echo "::endgroup::"
done <<< "$modules"
