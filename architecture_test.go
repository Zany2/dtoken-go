package dtoken_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// TestHTTPDependsOnContracts prevents concrete capabilities or global state from leaking back into HTTP adapters. TestHTTPDependsOnContracts 防止能力实现或全局状态重新进入 HTTP 适配层依赖。
func TestHTTPDependsOnContracts(t *testing.T) {
	checkArchitectureImports(t, []string{"httpauth", "integrations"}, func(path string) bool {
		for _, forbidden := range []string{"nonce", "oauth2", "ticket", "shortkey", "global"} {
			if matchesPackage(path, "github.com/Zany2/dtoken-go/"+forbidden) {
				return false
			}
		}
		return true
	})
}

// TestDomainDependencies keeps contracts, engines, and independent services free of implementation wiring. TestDomainDependencies 保持契约、引擎及独立服务不依赖实现装配层。
func TestDomainDependencies(t *testing.T) {
	for _, rule := range []struct {
		name        string
		directories []string
		allowed     []string
	}{
		{"contracts", []string{"contract"}, []string{"contract", "config", "errors"}},
		{"engine", []string{"internal/engine"}, []string{"contract", "config", "errors", "event", "internal/utils"}},
		{"capabilities", []string{"nonce", "oauth2", "ticket", "shortkey"}, []string{"contract", "config", "errors", "event", "internal/utils"}},
	} {
		t.Run(rule.name, func(t *testing.T) {
			checkArchitectureImports(t, rule.directories, func(path string) bool {
				const module = "github.com/Zany2/dtoken-go"
				if !matchesPackage(path, module) {
					return true
				}
				for _, allowed := range rule.allowed {
					if matchesPackage(path, module+"/"+allowed) {
						return true
					}
				}
				return false
			})
		})
	}
}

// matchesPackage includes subpackages without matching unrelated package-name prefixes. matchesPackage 包含子包但不匹配仅名称前缀相同的其他包。
func matchesPackage(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

// checkArchitectureImports inspects production imports including separately versioned integrations when present. checkArchitectureImports 检查生产代码依赖，并在目录存在时包含独立版本的框架集成。
func checkArchitectureImports(t *testing.T, directories []string, allowed func(string) bool) {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate source root")
	}
	root := filepath.Dir(source)
	for _, dir := range directories {
		// Published root modules exclude separately versioned integration directories. 发布的根模块不包含独立版本的集成目录。
		if dir == "integrations" {
			if _, err := os.Stat(filepath.Join(root, dir)); os.IsNotExist(err) {
				continue
			}
		}
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, spec := range file.Imports {
				imported, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					return err
				}
				if !allowed(imported) {
					t.Errorf("%s imports forbidden dependency %s", path, imported)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
