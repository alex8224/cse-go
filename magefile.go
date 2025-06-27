//go:build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Default = Build

const (
	// 定义构建输出目录
	buildDir = "./build"
)

// components 定义了所有需要独立编译的组件列表
// 如果未来有新的组件，只需在此处添加其名称即可
var components = []string{
	"printer",
}

// Build 编译 Supervisor 和所有的功能组件。这是默认任务。
func Build() {
	mg.Deps(BuildSupervisor, BuildComponents) // 使用 mg.Deps 来并行执行依赖任务
}

// BuildSupervisor 编译主应用程序 Supervisor
func BuildSupervisor() error {
	fmt.Println("--- Building Supervisor ---")
	return build("supervisor", "./cmd/supervisor/main.go")
}

// BuildComponents 编译在 components 变量中定义的所有组件
func BuildComponents() error {
	fmt.Println("--- Building Components ---")
	for _, component := range components {
		// 使用 mg.Deps 在循环中并行编译每一个组件
		err := build(component, fmt.Sprintf("./cmd/components/%s/main.go", component))
		if err != nil {
			return err
		}
	}
	return nil
}

// Run 编译并启动整个应用程序
func Run() error {
	mg.Deps(Build) // 确保在运行前所有内容都已编译
	fmt.Println("--- Starting Supervisor ---")
	// 执行编译好的 supervisor 程序
	return sh.Run(filepath.Join(buildDir, executableName("supervisor")))
}

// Clean 删除所有构建产物
func Clean() {
	fmt.Println("--- Cleaning build artifacts ---")
	os.RemoveAll(buildDir)
}

func build(name, path string) error {
	fmt.Printf("Building %s...\n", name)
	// 使用 sh.RunV 来执行 `go build` 命令，sh.RunV 会将命令的输出流式传输到控制台
	return sh.RunV("go", "build", "-v", "-o", filepath.Join(buildDir, executableName(name)), path)
}

// executableName 根据当前操作系统返回正确的可执行文件名 (例如，在 windows 上添加 .exe)
func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
