package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSSOModuleEntriesWithoutWorkspace checks minimum-Go discovery without loading go.work or invoking Go. TestSSOModuleEntriesWithoutWorkspace 检查最低 Go 版本模块发现，不加载 go.work 或调用 Go。
func TestSSOModuleEntriesWithoutWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "source with spaces")
	paths := []string{".", "sso", "storage/redis", "sso/storage/redis"}
	for _, path := range paths {
		directory := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
		module := "github.com/Zany2/dtoken-go"
		if path != "." {
			module += "/" + path
		}
		if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module "+module+"\r\n\r\ngo 1.25.0\r\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// A deliberately unusable workspace must not influence release-module discovery. 故意无效的工作区不得影响发布模块发现。
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("invalid workspace"), 0600); err != nil {
		t.Fatal(err)
	}
	entries, err := ssoModuleEntries(root)
	if err != nil || len(entries) != len(paths) {
		t.Fatalf("discovery: %v, %v", entries, err)
	}
	for i, entry := range entries {
		_, directory, ok := strings.Cut(entry, "=")
		if !ok || directory != filepath.Join(root, filepath.FromSlash(paths[i])) {
			t.Fatalf("invalid directory: %q", entry)
		}
	}
	manifest := filepath.Join(root, "sso", "go.mod")
	for _, content := range []string{"go 1.25.0\n", "module example.com/wrong\n", "module github.com/Zany2/dtoken-go/sso\nmodule github.com/Zany2/dtoken-go/sso\n"} {
		if err := os.WriteFile(manifest, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := ssoModuleEntries(root); err == nil {
			t.Fatal("invalid release manifest accepted")
		}
	}
	if err := os.Remove(manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := ssoModuleEntries(root); err == nil {
		t.Fatal("missing release module accepted")
	}
}
