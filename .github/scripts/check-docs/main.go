// Command check-docs compiles marked documentation examples without running them. 命令 check-docs 编译带标记的文档示例，不执行示例。
package main

import (
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var documents = []string{
	"README.md",
	"README_zh.md",
	"docs/tutorial/quick-start.md",
	"docs/tutorial/quick-start_zh.md",
	"docs/guide/core/authentication.md",
	"docs/guide/core/authentication_zh.md",
	"docs/guide/core/permission.md",
	"docs/guide/core/permission_zh.md",
	"docs/guide/integration/framework-integration.md",
	"docs/guide/integration/framework-integration_zh.md",
}

// Repeated names join initialization and usage fragments within one document. 同一文档中的重复名称连接初始化与使用片段。
var markedBlock = regexp.MustCompile("(?s)<!-- dtoken-compile: ([a-z0-9-]+) -->\\s*```go\\r?\\n(.*?)\\r?\\n```")

func main() {
	if err := checkDocs(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func checkDocs() error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	directory, err := os.MkdirTemp("", "dtoken-documentation-")
	if err != nil {
		return err
	}

	// Only clean up the temporary directory allocated above. 仅清理上面创建的临时目录。
	defer os.RemoveAll(directory)
	index := 0
	for _, document := range documents {
		data, err := os.ReadFile(filepath.Join(root, document))
		if err != nil {
			return err
		}
		programs, err := extractPrograms(string(data))
		if err != nil {
			return fmt.Errorf("%s: %w", document, err)
		}
		names := make([]string, 0, len(programs))
		for name := range programs {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			source, err := format.Source([]byte(programs[name]))
			if err != nil {
				return fmt.Errorf("%s (%s): %w", document, name, err)
			}
			example := filepath.Join(directory, fmt.Sprintf("example%d", index))
			index++
			if err := os.Mkdir(example, 0700); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(example, "main.go"), source, 0600); err != nil {
				return err
			}
			fmt.Printf("%s: %s (%s)\n", filepath.Base(example), document, name)
		}
	}
	env := append(os.Environ(), "GOWORK=off")
	commands := [][]string{
		{"mod", "init", "example.com/dtoken-docs"},
		{"mod", "edit", "-require=github.com/Zany2/dtoken-go@v0.0.0", "-require=github.com/Zany2/dtoken-go/integrations/gin@v0.0.0", "-replace=github.com/Zany2/dtoken-go=" + root, "-replace=github.com/Zany2/dtoken-go/integrations/gin=" + filepath.Join(root, "integrations", "gin")},
		{"mod", "tidy"},
		{"test", "-mod=readonly", "-run=^$", "./..."},
	}
	for _, args := range commands {
		command := exec.Command("go", args...)
		command.Dir, command.Env = directory, env
		command.Stdout, command.Stderr = os.Stdout, os.Stderr
		if err := command.Run(); err != nil {
			return fmt.Errorf("documentation compilation: go %s: %w", strings.Join(args, " "), err)
		}
	}
	return nil
}

func extractPrograms(document string) (map[string]string, error) {
	matches := markedBlock.FindAllStringSubmatch(document, -1)

	// Missing or malformed markers must not silently reduce coverage. 缺失或格式错误的标记不得静默减少检查覆盖。
	if len(matches) == 0 || len(matches) != strings.Count(document, "<!-- dtoken-compile:") {
		return nil, fmt.Errorf("missing or malformed documentation compile marker")
	}
	programs := map[string]string{}
	for _, match := range matches {
		programs[match[1]] += match[2] + "\n"
	}
	return programs, nil
}
