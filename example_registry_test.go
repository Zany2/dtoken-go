package dtoken_test

import (
	"context"
	"fmt"

	"github.com/Zany2/dtoken-go"
)

// ExampleAuthType demonstrates automatic registration and shared instance/selector operations. ExampleAuthType 演示自动注册及实例与选择器共用引擎。
func ExampleAuthType() {
	defer dtoken.CloseAll()
	ctx := context.Background()
	builder := dtoken.NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false)
	userAuth, err := builder.Clone().AuthType("user").Build()
	if err != nil {
		fmt.Println(err)
		return
	}
	adminAuth, err := builder.Clone().AuthType("admin").Build()
	if err != nil {
		fmt.Println(err)
		return
	}

	token, err := userAuth.LoginID(ctx, "10001")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(dtoken.AuthType("user").IsLogin(ctx, token))
	fmt.Println(adminAuth.IsLogin(ctx, token))

	// Output:
	// true
	// false
}

// ExampleBuilder_BuildInstance demonstrates isolated instances with the same namespace. ExampleBuilder_BuildInstance 演示同名但独立的实例。
func ExampleBuilder_BuildInstance() {
	ctx := context.Background()
	builder := dtoken.NewBuilder().IsPrintBanner(false).IsLog(false).AutoRenew(false).AsyncEvent(false)
	first, err := builder.BuildInstance()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer first.Close()
	second, err := builder.BuildInstance()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer second.Close()

	token, err := first.LoginID(ctx, "10001")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(first.IsLogin(ctx, token))
	fmt.Println(second.IsLogin(ctx, token))

	// Output:
	// true
	// false
}
