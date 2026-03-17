package v1

import (
	"github.com/gogf/gf/v2/frame/g"
)

// HelloReq defines the hello request 定义 hello 请求
type HelloReq struct {
	g.Meta `path:"/hello" tags:"Hello" method:"get" summary:"You first hello api"`
}

// HelloRes defines the hello response 定义 hello 响应
type HelloRes struct {
	g.Meta `mime:"text/html" example:"string"`
}
