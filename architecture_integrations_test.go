package dtoken_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// TestFrameworkRequestAPICoverage keeps request operations available across every framework despite native context naming differences. TestFrameworkRequestAPICoverage 保证各框架提供对应请求操作，同时允许原生上下文命名差异。
func TestFrameworkRequestAPICoverage(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate source root")
	}
	root := filepath.Join(filepath.Dir(source), "integrations")
	if _, err := os.Stat(root); os.IsNotExist(err) {
		t.Skip("published root module excludes framework modules")
	} else if err != nil {
		t.Fatal(err)
	}

	frameworks := []string{"gin", "beego", "chi", "echo", "fiber", "gf", "hertz", "kratos"}
	coverage := make(map[string]map[string]bool)
	operations := make(map[string]bool)
	for _, framework := range frameworks {
		methods := make(map[string]bool)
		dir := filepath.Join(root, framework)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, entry.Name()), nil, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Recv != nil || !function.Name.IsExported() {
					continue
				}
				name := function.Name.Name
				operation := strings.TrimSuffix(strings.TrimSuffix(name, "ByContext"), "ByCtx")
				// Context lookup aliases differ by framework; compare authentication operations only. 上下文查询别名因框架而异，仅比较认证操作。
				if operation == name || operation == "GetDTokenContext" {
					continue
				}
				methods[operation] = true
				operations[operation] = true
			}
		}
		if len(methods) == 0 {
			t.Fatalf("%s has no request operations", framework)
		}
		coverage[framework] = methods
	}

	// Compare the union so an operation added outside Gin also requires corresponding framework entries. 比较操作并集，确保新增于 Gin 之外的操作也有其他框架对应入口。
	names := make([]string, 0, len(operations))
	for operation := range operations {
		names = append(names, operation)
	}
	sort.Strings(names)
	for _, framework := range frameworks {
		t.Run(framework, func(t *testing.T) {
			for _, operation := range names {
				if !coverage[framework][operation] {
					t.Errorf("missing request operation %s (ByContext or ByCtx)", operation)
				}
			}
		})
	}
}
