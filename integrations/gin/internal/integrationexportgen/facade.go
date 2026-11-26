package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const facadeFile = "context_forward_generated.go"

// facadeTarget describes only native context syntax; request behavior stays in each adapter. facadeTarget 仅描述原生上下文语法，请求行为仍由各适配器维护。
type facadeTarget struct {
	contextType string
	importSpec  string
	label       string
	standard    bool
}

var facadeTargets = map[string]facadeTarget{
	"beego":  {"*beegocontext.Context", `beegocontext "github.com/beego/beego/v2/server/web/context"`, "Beego", false},
	"chi":    {"context.Context", `"context"`, "Chi", true},
	"echo":   {"echo4.Context", `echo4 "github.com/labstack/echo/v4"`, "Echo", false},
	"fiber":  {"*gofiber.Ctx", `gofiber "github.com/gofiber/fiber/v2"`, "Fiber", false},
	"gf":     {"context.Context", `"context"`, "GF", true},
	"hertz":  {"*hertzapp.RequestContext", `hertzapp "github.com/cloudwego/hertz/pkg/app"`, "Hertz", false},
	"kratos": {"context.Context", `"context"`, "Kratos", true},
}

// facadeSources stores canonical functions and their import spellings. facadeSources 保存标准转发函数及其导入写法。
type facadeSources struct {
	functions map[string]string
	imports   map[string]string
}

// readFacades selects grouped forwarding only, preserving native helpers and namespace-aware queries. readFacades 仅选择分组转发，保留原生辅助方法与支持命名空间的查询。
func readFacades(directory string) (*facadeSources, error) {
	paths, err := filepath.Glob(filepath.Join(directory, "context*facade.go"))
	if err != nil {
		return nil, err
	}
	result := &facadeSources{functions: map[string]string{}, imports: map[string]string{}}
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, source, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		for _, spec := range file.Imports {
			name, err := importName(spec)
			if err != nil {
				return nil, err
			}
			var text bytes.Buffer
			if err := format.Node(&text, fset, spec); err != nil {
				return nil, err
			}
			if previous, ok := result.imports[name]; ok && previous != text.String() {
				return nil, fmt.Errorf("conflicting facade import %s", name)
			}
			result.imports[name] = text.String()
		}
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || !sharedFacade(fn) {
				continue
			}
			if _, duplicate := result.functions[fn.Name.Name]; duplicate {
				return nil, fmt.Errorf("duplicate canonical facade %s", fn.Name.Name)
			}
			var parameter bytes.Buffer
			if len(fn.Type.Params.List) == 0 {
				return nil, fmt.Errorf("missing context parameter in %s", fn.Name.Name)
			}
			first := fn.Type.Params.List[0]
			if err := format.Node(&parameter, fset, first.Type); err != nil {
				return nil, err
			}
			if len(first.Names) != 1 || first.Names[0].Name != "c" || parameter.String() != "*gin.Context" {
				return nil, fmt.Errorf("unsupported context parameter in %s: %s", fn.Name.Name, parameter.String())
			}
			start := fn.Pos()
			if fn.Doc != nil {
				start = fn.Doc.Pos()
			}
			result.functions[fn.Name.Name] = string(source[fset.Position(start).Offset:fset.Position(fn.End()).Offset])
		}
	}
	if len(result.functions) == 0 {
		return nil, fmt.Errorf("no canonical facades in %s", directory)
	}
	return result, nil
}

// sharedFacade excludes queries whose native signatures intentionally differ. sharedFacade 排除原生签名有意不同的查询。
func sharedFacade(fn *ast.FuncDecl) bool {
	if fn.Recv != nil || fn.Body == nil || !ast.IsExported(fn.Name.Name) || !strings.HasSuffix(fn.Name.Name, "ByContext") {
		return false
	}
	switch fn.Name.Name {
	case "GetLoginIDByContext", "GetTokenInfoByContext", "IntrospectTokenByContext":
		return false
	}
	found := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		call, ok := selector.X.(*ast.CallExpr)
		if !ok || len(call.Args) != 0 {
			return true
		}
		group, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		receiver, ok := group.X.(*ast.Ident)
		if ok && receiver.Name == "dCtx" {
			found = true
		}
		return true
	})
	return found
}

// renderFacades maps native syntax while retaining canonical signatures, guards and business calls. renderFacades 映射原生语法，保留标准签名、前置检查和业务调用。
func renderFacades(source *facadeSources, packageName string) ([]byte, error) {
	target, ok := facadeTargets[packageName]
	if !ok {
		return nil, fmt.Errorf("unknown facade target %s", packageName)
	}
	names := make([]string, 0, len(source.functions))
	for name := range source.functions {
		names = append(names, name)
	}
	sort.Strings(names)
	var body strings.Builder
	for _, name := range names {
		text := source.functions[name]
		text = strings.ReplaceAll(text, "c *gin.Context", "c "+target.contextType)
		text = strings.ReplaceAll(text, "Gin", target.label)
		if target.standard {
			text = strings.ReplaceAll(text, "ByContext", "ByCtx")
			text = strings.ReplaceAll(text, "requestContext(c)", "c")
		}
		body.WriteString(text + "\n\n")
	}
	imports := make(map[string]string, len(source.imports))
	for name, spec := range source.imports {
		if name != "gin" {
			imports[name] = spec
		}
	}
	imports[strings.TrimPrefix(strings.Split(target.contextType, ".")[0], "*")] = target.importSpec

	// Infer imports from the transformed declarations instead of duplicating per-framework lists. 从转换后的声明推导导入，避免维护每个框架的重复列表。
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, facadeFile, "package "+packageName+"\n"+body.String(), parser.ParseComments)
	if err != nil {
		return nil, err
	}
	if target.standard {
		// Rename identifiers through the AST so literal values and comments remain unchanged. 通过语法树重命名标识符，避免修改字面值与注释。
		ast.Inspect(file, func(node ast.Node) bool {
			if identifier, ok := node.(*ast.Ident); ok && identifier.Name == "c" {
				identifier.Name = "ctx"
			}
			return true
		})
	}
	var declarations bytes.Buffer
	if err := format.Node(&declarations, fset, file); err != nil {
		return nil, err
	}
	text := strings.TrimPrefix(declarations.String(), "package "+packageName+"\n")
	used := selectorNames(file)
	var specs []string
	for name, spec := range imports {
		if used[name] {
			specs = append(specs, spec)
		}
	}
	sort.Strings(specs)
	return format.Source([]byte(generatedHeader + "package " + packageName + "\n\nimport (\n" + strings.Join(specs, "\n") + "\n)\n\n" + text))
}

// stripFacades removes generated declarations only and keeps native code and comments in place. stripFacades 仅移除转为生成的声明，保留原生代码及注释。
func stripFacades(source []byte, names map[string]string) ([]byte, bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "native.go", source, parser.ParseComments)
	if err != nil {
		return nil, false, err
	}
	type span struct{ start, end int }
	var removed []span
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		name := strings.TrimSuffix(fn.Name.Name, "ByCtx")
		if name != fn.Name.Name {
			name += "ByContext"
		}
		if _, shared := names[name]; !shared {
			continue
		}
		start := fn.Pos()
		if fn.Doc != nil {
			start = fn.Doc.Pos()
		}
		removed = append(removed, span{fset.Position(start).Offset, fset.Position(fn.End()).Offset})
	}
	if len(removed) == 0 {
		return source, false, nil
	}
	for i := len(removed) - 1; i >= 0; i-- {
		r := removed[i]
		source = append(source[:r.start:r.start], source[r.end:]...)
	}
	fset = token.NewFileSet()
	file, err = parser.ParseFile(fset, "native.go", source, parser.ParseComments)
	if err != nil {
		return nil, false, err
	}
	used := selectorNames(file)
	var declarations []ast.Decl
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.IMPORT {
			declarations = append(declarations, declaration)
			continue
		}
		var specs []ast.Spec
		for _, spec := range group.Specs {
			name, err := importName(spec.(*ast.ImportSpec))
			if err != nil {
				return nil, false, err
			}
			if used[name] || name == "_" || name == "." {
				specs = append(specs, spec)
			}
		}
		if len(specs) > 0 {
			group.Specs = specs
			declarations = append(declarations, group)
		}
	}
	file.Decls = declarations
	if len(declarations) == 0 {
		return nil, true, nil
	}
	var output bytes.Buffer
	if err := format.Node(&output, fset, file); err != nil {
		return nil, false, err
	}
	return output.Bytes(), true, nil
}

func importName(spec *ast.ImportSpec) (string, error) {
	if spec.Name != nil {
		return spec.Name.Name, nil
	}
	path, err := strconv.Unquote(spec.Path.Value)
	return filepath.Base(path), err
}

func selectorNames(node ast.Node) map[string]bool {
	used := map[string]bool{}
	ast.Inspect(node, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok {
			if name, ok := selector.X.(*ast.Ident); ok {
				used[name.Name] = true
			}
		}
		return true
	})
	return used
}

// syncFacades validates all transformations before migrating duplicated declarations. syncFacades 在迁移重复声明前校验全部转换结果。
func syncFacades(directory string, source *facadeSources, packageName string, checkOnly bool) error {
	output, err := renderFacades(source, packageName)
	if err != nil {
		return err
	}
	paths, err := filepath.Glob(filepath.Join(directory, "*.go"))
	if err != nil {
		return err
	}
	changes := map[string][]byte{}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") || filepath.Base(path) == facadeFile {
			continue
		}
		current, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		cleaned, changed, err := stripFacades(current, source.functions)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if changed {
			if checkOnly {
				return fmt.Errorf("handwritten forwarding remains in %s; run go generate in integrations/gin", path)
			}
			changes[path] = cleaned
		}
	}
	if err := writeOrCheck(filepath.Join(directory, facadeFile), output, checkOnly); err != nil {
		return err
	}
	// Keep migration writes deterministic for review and repeatability. 保持迁移写入顺序确定，便于审查和重复执行。
	var changedPaths []string
	for path := range changes {
		changedPaths = append(changedPaths, path)
	}
	sort.Strings(changedPaths)
	for _, path := range changedPaths {
		content := changes[path]
		if content == nil {
			err = os.Remove(path)
		} else {
			err = os.WriteFile(path, content, 0o644)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
