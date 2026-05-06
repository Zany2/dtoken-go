package hello

import (
	"gf_v2/api/hello"
)

// ControllerV1 implements the v1 hello controller 实现 v1 hello 控制器
type ControllerV1 struct{}

// NewV1 creates the v1 hello controller 创建 v1 hello 控制器
func NewV1() hello.IHelloV1 {
	return &ControllerV1{}
}
