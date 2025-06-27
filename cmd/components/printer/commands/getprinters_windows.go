//go:build windows

// Package commands 包含了打印组件特定的命令实现
package commands

import (
	"encoding/json"
	"fmt"
	"log"

	"cse-go/internal/commandbus"
	pb "cse-go/pkg/api/v1"

	winprinter "github.com/godoes/printers"
)

// 确保 GetPrintersCmd 实现了 commandbus.Command 接口
var _ commandbus.Command = (*GetPrintersCmd)(nil)

// GetPrintersCmd 结构体定义
type GetPrintersCmd struct{}

// Name 返回命令名称
func (c *GetPrintersCmd) Name() string {
	return "print.getPrinters"
}

// GetInfo 返回命令元数据
func (c *GetPrintersCmd) GetInfo() *pb.CommandInfo {
	return &pb.CommandInfo{
		CommandName:      c.Name(),
		Description:      "获取所有可用的打印机列表 (Windows)。",
		ParametersSchema: `{}`,
		ResultSchema:     `{"type": "array", "items": {"type": "string"}}`,
	}
}

// Execute 获取所有可用的打印机列表
func (c *GetPrintersCmd) Execute(params *pb.CommandParams) (*pb.CommandResult, error) {
	// 获取所有打印机名称列表
	printerNames, err := winprinter.ReadNames()
	if err != nil {
		log.Printf("获取打印机列表失败: %v", err)
		// 如果获取失败，返回空列表而不是错误，保证系统稳定性
		emptyList, _ := json.Marshal([]string{})
		return &pb.CommandResult{
			JsonPayload: string(emptyList),
		}, nil
	}

	// 将打印机列表序列化为 JSON
	printerListJson, err := json.Marshal(printerNames)
	if err != nil {
		log.Printf("序列化打印机列表失败: %v", err)
		// 序列化失败时返回空列表
		emptyList, _ := json.Marshal([]string{})
		return &pb.CommandResult{
			JsonPayload: string(emptyList),
		}, nil
	}

	fmt.Printf("成功获取到 %d 个打印机\n", len(printerNames))
	return &pb.CommandResult{
		JsonPayload: string(printerListJson),
	}, nil
}

// init 自动注册命令
func init() {
	GlobalRegistry.Register(&GetPrintersCmd{})
}
