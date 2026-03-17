package main

import (
	_ "gf_v2/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"gf_v2/internal/cmd"
)

// main starts the GoFrame v2 example 启动 GoFrame v2 示例
func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
