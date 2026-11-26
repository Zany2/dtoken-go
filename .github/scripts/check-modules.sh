#!/usr/bin/env bash
set -euo pipefail

# Run the portable checker from the checkout root. 从仓库根目录运行跨平台检查器。
cd "$(dirname "${BASH_SOURCE[0]}")/../.."
go run ./.github/scripts/check-modules "$@"
