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

// TestHTTPDependsOnContracts keeps concrete capabilities, component wiring, and peer frameworks out of HTTP adapters. TestHTTPDependsOnContracts 防止能力实现、组件装配和其他框架进入 HTTP 适配层依赖。
func TestHTTPDependsOnContracts(t *testing.T) {
	checkArchitectureImports(t, []string{"httpauth", "integrations"}, func(source, path string) bool {
		const module = "github.com/Zany2/dtoken-go"

		// Adapters may organize their own helpers in subpackages, but cannot depend on another framework. 适配器可以用子包组织自身辅助代码，但不能依赖其他框架。
		if matchesPackage(path, module+"/integrations") {
			parts := strings.Split(source, "/")
			return len(parts) > 2 && parts[0] == "integrations" &&
				matchesPackage(path, module+"/integrations/"+parts[1])
		}

		for _, forbidden := range []string{
			"nonce", "oauth2", "ticket", "shortkey", "global", "sso",
			"codec", "generator", "logging", "pool", "storage",
			"internal/builder", "examples", "tests",
			"contract/storagetest",
		} {
			if matchesPackage(path, module+"/"+forbidden) {
				return false
			}
		}
		return true
	})
}

// TestDomainDependencies enforces direct repository import boundaries for foundations, components, assembly, and authentication. TestDomainDependencies 约束基础包、组件、装配和认证代码的仓库内直接导入边界。
func TestDomainDependencies(t *testing.T) {
	for _, rule := range []struct {
		name        string
		directories []string
		allowed     []string
	}{
		{"contracts", []string{"contract"}, []string{"contract", "errors"}},
		{"config", []string{"config"}, []string{"config", "contract", "errors"}},
		{"errors", []string{"errors"}, []string{"errors"}},
		{"events", []string{"event"}, []string{"event", "contract", "errors"}},
		{"utils", []string{"internal/utils"}, []string{"internal/utils"}},
		{"banner", []string{"internal/banner"}, []string{"internal/banner", "config", "contract"}},
		{"engine", []string{"internal/engine"}, []string{"contract", "config", "errors", "event", "internal/utils"}},
		{"capabilities", []string{"nonce", "oauth2", "ticket", "shortkey"}, []string{"contract", "config", "errors", "event", "internal/utils"}},
		{"http", []string{"httpauth"}, []string{"httpauth", "contract", "config", "errors", "event", "internal/engine"}},
		{"builder", []string{"internal/builder"}, []string{
			"internal/builder", "internal/engine", "internal/banner", "internal/utils", "contract", "config", "errors",
			"codec/json", "generator/dgenerator", "logging/dlog", "pool/ants", "storage/memory",
		}},
		{"codecs", []string{"codec"}, []string{"codec", "contract", "config", "errors", "internal/utils"}},
		{"generators", []string{"generator"}, []string{"generator", "contract", "config", "errors", "internal/utils"}},
		{"logging", []string{"logging"}, []string{"logging", "contract", "config", "errors", "internal/utils"}},
		{"pools", []string{"pool"}, []string{"pool", "contract", "config", "errors", "internal/utils"}},
		{"storage", []string{"storage"}, []string{"storage", "contract", "config", "errors", "internal/utils"}},
	} {
		t.Run(rule.name, func(t *testing.T) {
			checkArchitectureImports(t, rule.directories, func(_ string, path string) bool {
				const module = "github.com/Zany2/dtoken-go"
				if !matchesPackage(path, module) {
					return true
				}

				// Storage contract helpers are for tests, even though their implementation uses regular Go files. 存储契约辅助包仅用于测试，即使其实现使用普通 Go 文件。
				if matchesPackage(path, module+"/contract/storagetest") {
					return false
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

// checkArchitectureImports checks imports with repository-relative source paths, including separately versioned integrations when present. checkArchitectureImports 使用仓库相对源码路径检查导入，并在目录存在时包含独立版本的框架集成。
func checkArchitectureImports(t *testing.T, directories []string, allowed func(source, imported string) bool) {
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
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			for _, spec := range file.Imports {
				imported, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					return err
				}
				if !allowed(filepath.ToSlash(relative), imported) {
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
