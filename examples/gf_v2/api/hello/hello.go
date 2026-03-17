// -------------------------------------------------- Generated API - 生成接口请勿手动编辑 --------------------------------------------------

package hello

import (
	"context"

	"gf_v2/api/hello/v1"
)

// IHelloV1 defines the hello v1 API 定义 hello v1 接口
type IHelloV1 interface {
	Hello(ctx context.Context, req *v1.HelloReq) (res *v1.HelloRes, err error)
}
