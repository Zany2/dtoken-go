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
	"github.com/Zany2/dtoken-go/com/log/dlog"
)

type (
	Logger                     = dlog.Logger
)

var (
	NewLoggerWithConfig            = dlog.NewLoggerWithConfig
)
`

// TestExportFileSpecs verifies only truly shared exports are generated. TestExportFileSpecs 验证仅生成真正同构的导出文件。
func TestExportFileSpecs(t *testing.T) {
	expected := map[string]bool{
		"api_export.go":       false,
		"component_export.go": true,
		"error_export.go":     false,
		"facade_export.go":    false,
	}
	if len(exportFiles) != len(expected) {
		t.Fatalf("export file count = %d, want %d", len(exportFiles), len(expected))
	}

	for _, exportFile := range exportFiles {
		includeGFLogger, ok := expected[exportFile.name]
		if !ok {
			t.Fatalf("unexpected generated export file %q", exportFile.name)
		}
		if exportFile.includeGFLogger != includeGFLogger {
			t.Fatalf("%s includeGFLogger = %v, want %v", exportFile.name, exportFile.includeGFLogger, includeGFLogger)
		}
	}
}

// TestRenderCommonFramework verifies package replacement without GoFrame extensions. TestRenderCommonFramework 验证普通框架只替换包名且不注入 GoFrame 扩展。
func TestRenderCommonFramework(t *testing.T) {
	output, err := render([]byte(testSource), targetSpec{packageName: "echo"})
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

// TestRenderGoFrameFramework verifies GoFrame logger extensions are injected once. TestRenderGoFrameFramework 验证 GoFrame 日志扩展仅注入一次。
func TestRenderGoFrameFramework(t *testing.T) {
	output, err := render([]byte(testSource), targetSpec{packageName: "gf", includeGFLogger: true})
	if err != nil {
		t.Fatalf("render() error = %v", err)
	}

	text := string(output)
	for _, expected := range []string{
		`gflog "github.com/Zany2/dtoken-go/com/log/gf"`,
		"gflog.GFLogger",
		"gflog.NewGFLogger",
	} {
		if strings.Count(text, expected) != 1 {
			t.Fatalf("generated output count for %q is not 1:\n%s", expected, text)
		}
	}
}

// TestGeneratedFilesAreUpToDate verifies committed exports match the Gin canonical sources. TestGeneratedFilesAreUpToDate 验证已提交导出文件与 Gin 标准源保持同步。
func TestGeneratedFilesAreUpToDate(t *testing.T) {
	canonicalDir := filepath.Join("..", "..")
	integrationsDir := filepath.Join("..", "..", "..")

	for _, exportFile := range exportFiles {
		exportFile := exportFile
		t.Run(exportFile.name, func(t *testing.T) {
			sourcePath := filepath.Join(canonicalDir, exportFile.name)
			source, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("read canonical source %q: %v", sourcePath, err)
			}

			for _, packageName := range targetPackages {
				packageName := packageName
				t.Run(packageName, func(t *testing.T) {
					expected, err := render(source, targetSpec{
						packageName:     packageName,
						includeGFLogger: exportFile.includeGFLogger && packageName == "gf",
					})
					if err != nil {
						t.Fatalf("render %s: %v", packageName, err)
					}

					targetPath := filepath.Join(integrationsDir, packageName, exportFile.name)
					actual, err := os.ReadFile(targetPath)
					if err != nil {
						t.Fatalf("read generated file %q: %v", targetPath, err)
					}
					if !bytes.Equal(normalizeLineEndings(actual), normalizeLineEndings(expected)) {
						t.Fatalf("generated file %q is out of date; run go generate in integrations/gin", targetPath)
					}
				})
			}
		})
	}
}

func normalizeLineEndings(data []byte) []byte {
	return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
}
