package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFacadeTargetsPreserveForwarding verifies native context types, error guards and variadic calls. TestFacadeTargetsPreserveForwarding 验证原生上下文类型、错误检查与可变参数调用。
func TestFacadeTargetsPreserveForwarding(t *testing.T) {
	source := &facadeSources{
		imports: map[string]string{"gin": `"github.com/gin-gonic/gin"`},
		functions: map[string]string{
			"LoginByContext": `// LoginByContext logs in a Gin request. LoginByContext 登录 Gin 请求。
func LoginByContext(c *gin.Context, loginID string, device ...string) (string, error) {
	dCtx, err := requireDTokenContextByContext(c)
	if err != nil { return "", err }
	return dCtx.Auth().Login(requestContext(c), loginID, device...)
}`,
		},
	}
	for _, name := range targetPackages {
		t.Run(name, func(t *testing.T) {
			output, err := renderFacades(source, name)
			if err != nil {
				t.Fatal(err)
			}
			text := string(output)
			parameter := "c "
			if facadeTargets[name].standard {
				parameter = "ctx "
			}
			if !strings.Contains(text, parameter+facadeTargets[name].contextType) || strings.Contains(text, "gin.Context") {
				t.Fatalf("wrong native context:\n%s", text)
			}
			call := "dCtx.Auth().Login(requestContext(c), loginID, device...)"
			if facadeTargets[name].standard {
				call = "dCtx.Auth().Login(ctx, loginID, device...)"
				if !strings.Contains(text, "requireDTokenContextByCtx(ctx)") || strings.Contains(text, "requestContext(") {
					t.Fatalf("standard context path changed:\n%s", text)
				}
			}
			if !strings.Contains(text, call) || !strings.Contains(text, `return "", err`) {
				t.Fatalf("forwarding or error result changed:\n%s", text)
			}
		})
	}
}

// TestCanonicalFacadesRender verifies real imports, API selection and parseable output for every target. TestCanonicalFacadesRender 验证真实导入、API 选择以及每个目标输出均可解析。
func TestCanonicalFacadesRender(t *testing.T) {
	source, err := readFacades(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"LoginByContext", "ValidateTicketByContext", "SetTokenCookieByContext"} {
		if _, ok := source.functions[name]; !ok {
			t.Fatalf("missing shared facade %s", name)
		}
	}
	for _, name := range []string{"GetLoginIDByContext", "GetTokenInfoByContext", "IntrospectTokenByContext", "GetManagerByContext"} {
		if _, ok := source.functions[name]; ok {
			t.Fatalf("native operation selected: %s", name)
		}
	}
	for _, name := range targetPackages {
		output, err := renderFacades(source, name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		file, err := parser.ParseFile(token.NewFileSet(), facadeFile, output, parser.AllErrors)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, declaration := range file.Decls {
			if _, ok := declaration.(*ast.FuncDecl); ok {
				count++
			}
		}
		if count != len(source.functions) {
			t.Fatalf("%s lost declarations: %d != %d", name, count, len(source.functions))
		}
	}
}

// TestFacadeMigrationPreservesNativeCode verifies only shared declarations and unused imports are removed. TestFacadeMigrationPreservesNativeCode 验证仅移除共享声明及不再使用的导入。
func TestFacadeMigrationPreservesNativeCode(t *testing.T) {
	source := []byte(`package chi
import (
	"context"
	"time"
)
// LoginByCtx is migrated. LoginByCtx 将迁移。
func LoginByCtx(ctx context.Context, timeout time.Duration) {}
// GetLoginIDByCtx keeps native namespace selection. GetLoginIDByCtx 保留原生命名空间选择。
func GetLoginIDByCtx(ctx context.Context, authType ...string) (string, error) { return "native", nil }
// keep stays unchanged. keep 保持不变。
func keep(ctx context.Context) { _ = ctx }
`)
	names := map[string]string{"LoginByContext": ""}
	output, changed, err := stripFacades(source, names)
	if err != nil || !changed {
		t.Fatalf("migration: changed=%v, err=%v", changed, err)
	}
	text := string(output)
	if strings.Contains(text, "func LoginByCtx") || strings.Contains(text, `"time"`) || strings.Contains(text, "is migrated") {
		t.Fatalf("old declaration or import remains:\n%s", text)
	}
	if !strings.Contains(text, "func keep(ctx context.Context)") || !strings.Contains(text, "authType ...string") || !strings.Contains(text, "keeps native namespace") {
		t.Fatalf("native behavior removed:\n%s", text)
	}
	second, changed, err := stripFacades(output, names)
	if err != nil || changed || !bytes.Equal(output, second) {
		t.Fatalf("migration is not idempotent: changed=%v, err=%v", changed, err)
	}
	output, changed, err = stripFacades([]byte("package chi\nfunc LoginByCtx() {}\n"), names)
	if err != nil || !changed || output != nil {
		t.Fatalf("empty source not removed: %q, %v, %v", output, changed, err)
	}
}

// TestFacadeCheckDoesNotMigrate verifies check mode leaves handwritten files untouched. TestFacadeCheckDoesNotMigrate 验证检查模式不会迁移手写文件。
func TestFacadeCheckDoesNotMigrate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "context_facade.go")
	original := []byte("package chi\nfunc LoginByCtx() {}\n")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	source := &facadeSources{
		functions: map[string]string{"LoginByContext": "func LoginByContext(c *gin.Context) {}"},
	}
	if err := syncFacades(dir, source, "chi", true); err == nil {
		t.Fatal("handwritten facade passed generated check")
	}
	current, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(original, current) {
		t.Fatalf("check mode changed native source: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, facadeFile)); !os.IsNotExist(err) {
		t.Fatalf("check mode created generated output: %v", err)
	}

	if err := syncFacades(dir, source, "chi", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("empty handwritten file remains: %v", err)
	}
	if err := syncFacades(dir, source, "chi", true); err != nil {
		t.Fatalf("generated output failed check: %v", err)
	}
	first, err := os.ReadFile(filepath.Join(dir, facadeFile))
	if err != nil {
		t.Fatal(err)
	}
	if err := syncFacades(dir, source, "chi", false); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(dir, facadeFile))
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("repeated generation changed output: %v", err)
	}
}
