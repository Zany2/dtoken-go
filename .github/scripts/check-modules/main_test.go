package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCopyModuleExcludesWorkspaceFiles verifies copies cannot see sibling modules or checkout metadata. TestCopyModuleExcludesWorkspaceFiles 验证副本不包含嵌套模块、工作区文件和仓库元数据。
func TestCopyModuleExcludesWorkspaceFiles(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source with spaces")
	for _, path := range []string{"go.mod", "go.work", "go.work.sum", "main.go", "testdata/input.json", "nested/go.mod", "nested/hidden.go", ".git/config"} {
		full := filepath.Join(source, path)
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("content"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	destination := filepath.Join(t.TempDir(), "module")
	if err := copyModule(source, destination); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"go.mod", "main.go", "testdata/input.json"} {
		data, err := os.ReadFile(filepath.Join(destination, path))
		if err != nil || string(data) != "content" {
			t.Fatalf("missing module file %s: %v", path, err)
		}
	}
	for _, path := range []string{"nested", ".git", "go.work", "go.work.sum"} {
		if _, err := os.Stat(filepath.Join(destination, path)); !os.IsNotExist(err) {
			t.Fatalf("workspace file leaked into copy: %s (%v)", path, err)
		}
	}
}
