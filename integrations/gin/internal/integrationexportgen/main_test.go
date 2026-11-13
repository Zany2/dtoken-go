package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSource = `// source-only comment
//go:generate go run ./internal/integrationexportgen
package gin

import (
	"github.com/Zany2/dtoken-go"
)

type (
	Auth = dtoken.Auth
)

var (
	NewBuilder = dtoken.NewBuilder
)
`

// TestExportFileSpecs verifies only truly shared exports are generated. TestExportFileSpecs 验证仅生成真正同构的导出文件。
func TestExportFileSpecs(t *testing.T) {
	expected := []string{"error_export.go", "export.go"}
	if strings.Join(exportFiles, ",") != strings.Join(expected, ",") {
		t.Fatalf("export files = %v, want %v", exportFiles, expected)
	}
}

// TestRenderCommonFramework verifies package replacement without GoFrame extensions. TestRenderCommonFramework 验证普通框架只替换包名且不注入 GoFrame 扩展。
func TestRenderCommonFramework(t *testing.T) {
	output, err := render([]byte(testSource), "echo")
	if err != nil {
		t.Fatalf("render() error = %v", err)
	}

	text := string(output)
	if !strings.HasPrefix(text, generatedHeader+"package echo\n") {
		t.Fatalf("generated header or package mismatch:\n%s", text)
	}
	if strings.Contains(text, "go:generate") || strings.Contains(text, "gflog") {
		t.Fatalf("common output contains source-only content:\n%s", text)
	}
}

// TestCanonicalSourcesRender verifies real sources render for every target without sibling modules. TestCanonicalSourcesRender 验证真实源文件能为所有框架生成导出，无需其他模块目录。
func TestCanonicalSourcesRender(t *testing.T) {
	for _, exportFile := range exportFiles {
		source, err := os.ReadFile(filepath.Join("..", "..", exportFile))
		if err != nil {
			t.Fatal(err)
		}
		for _, packageName := range targetPackages {
			if _, err := render(source, packageName); err != nil {
				t.Fatalf("%s/%s: %v", packageName, exportFile, err)
			}
		}
	}
}

// TestCheckModeRejectsDriftWithoutWriting verifies repository checks fail without repairing the evidence. TestCheckModeRejectsDriftWithoutWriting 验证仓库检查能拒绝差异且不修改文件。
func TestCheckModeRejectsDriftWithoutWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "export.go")
	expected := []byte("package echo\n")
	if err := writeOrCheck(path, expected, true); err == nil {
		t.Fatal("missing file passed")
	}
	actual := []byte("package stale\n")
	if err := os.WriteFile(path, actual, 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeOrCheck(path, expected, true); err == nil {
		t.Fatal("stale export passed")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, actual) {
		t.Fatal("check mode modified the export")
	}
	if err := os.WriteFile(path, []byte("package echo\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeOrCheck(path, expected, true); err != nil {
		t.Fatal(err)
	}
}
