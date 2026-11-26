package dtoken_test

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestIntegrationFacadeForwarding keeps grouped request operations consistent across native adapters. TestIntegrationFacadeForwarding 保证各原生适配器的分组请求操作转发一致。
func TestIntegrationFacadeForwarding(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate source root")
	}
	root := filepath.Join(filepath.Dir(source), "integrations")
	if _, err := os.Stat(root); os.IsNotExist(err) {
		t.Skip("published root modules do not include integration modules")
	}
	canonical := integrationForwarding(t, filepath.Join(root, "gin"))
	if len(canonical) == 0 {
		t.Fatal("no canonical request forwarding methods found")
	}
	for _, framework := range []string{"beego", "chi", "echo", "fiber", "gf", "hertz", "kratos"} {
		t.Run(framework, func(t *testing.T) {
			actual := integrationForwarding(t, filepath.Join(root, framework))
			for name, want := range canonical {
				if got := actual[name]; got != want {
					t.Errorf("%s forwards %q, want %q", name, got, want)
				}
			}
			for name := range actual {
				if _, exists := canonical[name]; !exists {
					t.Errorf("%s has no matching canonical request operation", name)
				}
			}
		})
	}
}

// integrationForwarding compares business destinations and arguments while preserving framework-specific context handling. integrationForwarding 比较业务目标及参数，保留各框架的上下文处理差异。
func integrationForwarding(t *testing.T, directory string) map[string]string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(directory, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	result := make(map[string]string)
	fset := token.NewFileSet()
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil || !ast.IsExported(fn.Name.Name) {
				continue
			}
			name := strings.TrimSuffix(strings.TrimSuffix(fn.Name.Name, "ByContext"), "ByCtx")
			if name == fn.Name.Name {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				method, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				group, ok := method.X.(*ast.CallExpr)
				if !ok || len(group.Args) != 0 {
					return true
				}
				selector, ok := group.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				receiver, ok := selector.X.(*ast.Ident)
				if !ok || receiver.Name != "dCtx" {
					return true
				}
				var args []string
				for index, arg := range call.Args {
					var formatted bytes.Buffer
					if err := format.Node(&formatted, fset, arg); err != nil {
						t.Fatal(err)
					}
					value := formatted.String()
					if index == 0 && (value == "requestContext(c)" || value == "requestContext(ctx)") {
						value = "ctx"
					}
					if index == len(call.Args)-1 && call.Ellipsis.IsValid() {
						value += "..."
					}
					args = append(args, value)
				}
				if _, duplicate := result[name]; duplicate {
					t.Fatalf("multiple grouped operations in %s", fn.Name.Name)
				}
				result[name] = selector.Sel.Name + "." + method.Sel.Name + "(" + strings.Join(args, ",") + ")"
				return false
			})
		}
	}
	return result
}
