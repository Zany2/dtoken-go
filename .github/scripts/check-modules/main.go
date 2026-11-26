// Command check-modules tests module copies outside the checkout with workspace resolution disabled. 命令 check-modules 在仓库外关闭工作区解析后测试模块副本。
package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if err := checkModules(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func checkModules(selected []string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}

	var modules []string
	if len(selected) > 0 && selected[0] == "-sso" {
		// Read the release dependency set without loading the newer workspace Go version. 不加载较新的工作区 Go 版本，直接读取发布依赖集合。
		selected = selected[1:]
		modules, err = ssoModuleEntries(root)
		if err != nil {
			return err
		}
		if len(selected) == 0 {
			selected = []string{"sso", "sso/storage/redis"}
		}
	} else {
		// Resolve checkout replacements before disabling the workspace. 关闭工作区前解析源码替换路径。
		list := exec.Command("go", "list", "-m", "-f", "{{.Path}}={{.Dir}}")
		list.Env = append(os.Environ(), "GOWORK="+filepath.Join(root, "go.work"))
		output, err := list.Output()
		if err != nil {
			return fmt.Errorf("list workspace modules: %w", err)
		}
		// Module paths can contain spaces on Windows; split by lines instead. Windows 模块目录可能包含空格，须按行拆分。
		modules = strings.Split(strings.TrimSpace(string(output)), "\n")
	}
	requested := map[string]bool{}
	for _, path := range selected {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		requested[absolute] = true
	}
	tempDir, err := os.MkdirTemp("", "dtoken-isolated-modules-")
	if err != nil {
		return err
	}

	// Only remove the directory created by this invocation. 仅清理本次调用创建的临时目录。
	defer os.RemoveAll(tempDir)
	for index, module := range modules {
		name, directory, ok := strings.Cut(strings.TrimSpace(module), "=")
		if !ok {
			return fmt.Errorf("invalid module entry: %q", module)
		}
		if len(selected) > 0 && !requested[directory] {
			continue
		}
		delete(requested, directory)
		fmt.Println("Checking isolated module:", name)
		isolated := filepath.Join(tempDir, fmt.Sprint(index), "module")
		if err := copyModule(directory, isolated); err != nil {
			return err
		}
		env := append(os.Environ(), "GOWORK=off")
		if strings.HasSuffix(name, "/codec/jsonv2") {
			env = append(env, "GOEXPERIMENT=jsonv2")
		}

		// Add local replacements only to the disposable module copy. 仅在临时模块副本中加入源码替换。
		args := []string{"mod", "edit"}
		for _, replacement := range modules {
			args = append(args, "-replace="+strings.TrimSpace(replacement))
		}
		if err := runGo(isolated, env, args...); err != nil {
			return err
		}

		// Reuse known checksums without rewriting the checkout's module files. 复用已有校验和，不改写仓库模块文件。
		rootSum, err := os.ReadFile(filepath.Join(root, "go.sum"))
		if err != nil {
			return err
		}
		sum, err := os.OpenFile(filepath.Join(isolated, "go.sum"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		_, writeErr := sum.Write(append([]byte("\n"), rootSum...))
		closeErr := sum.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
		if err := runGo(isolated, env, "vet", "-mod=readonly", "./..."); err != nil {
			return err
		}
		if err := runGo(isolated, env, "test", "-mod=readonly", "-count=1", "-timeout=5m", "./..."); err != nil {
			return err
		}
	}
	if len(requested) > 0 {
		return fmt.Errorf("requested paths are not workspace modules: %v", requested)
	}
	return nil
}

// ssoModuleEntries verifies the four local release modules without invoking workspace discovery. ssoModuleEntries 校验四个本地发布模块，不调用工作区发现。
func ssoModuleEntries(root string) ([]string, error) {
	const base = "github.com/Zany2/dtoken-go"
	modules := make([]string, 0, 4)
	for _, relative := range []string{".", "sso", "storage/redis", "sso/storage/redis"} {
		directory := filepath.Join(root, filepath.FromSlash(relative))
		data, err := os.ReadFile(filepath.Join(directory, "go.mod"))
		if err != nil {
			return nil, err
		}
		expected := base
		if relative != "." {
			expected += "/" + relative
		}
		found := false
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[0] == "module" {
				if found || strings.Trim(fields[1], "\"") != expected {
					return nil, fmt.Errorf("%s: expected module %s", relative, expected)
				}
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("%s: missing module declaration", relative)
		}
		modules = append(modules, expected+"="+directory)
	}
	return modules, nil
}

// copyModule excludes nested modules and checkout metadata, matching module isolation. copyModule 排除嵌套模块及仓库元数据，保证模块目录隔离。
func copyModule(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != source {
				if strings.HasPrefix(entry.Name(), ".") {
					return filepath.SkipDir
				}
				if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
					return filepath.SkipDir
				} else if !os.IsNotExist(err) {
					return err
				}
			}
			return os.MkdirAll(filepath.Join(destination, relative), 0700)
		}
		if !entry.Type().IsRegular() || entry.Name() == "go.work" || entry.Name() == "go.work.sum" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(destination, relative), data, info.Mode().Perm())
	})
}

func runGo(directory string, env []string, args ...string) error {
	command := exec.Command("go", args...)
	command.Dir, command.Env = directory, env
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s: go %s: %w", directory, strings.Join(args, " "), err)
	}
	return nil
}
