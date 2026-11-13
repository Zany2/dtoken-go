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
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate source root")
	}
	root := filepath.Dir(source)
	forbidden := map[string]bool{}
	for _, pkg := range []string{"nonce", "oauth2", "ticket", "shortkey", "global"} {
		forbidden["github.com/Zany2/dtoken-go/"+pkg] = true
	}
	for _, dir := range []string{"httpauth", "integrations", "contract"} {
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
				if forbidden[imported] {
					t.Errorf("%s imports implementation %s", path, imported)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
