package main

import (
	_ "gf_v2/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"gf_v2/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
