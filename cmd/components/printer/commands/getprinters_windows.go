//go:build windows

// Package commands 包含了打印组件特定的命令实现
package commands

import (
	"encoding/json"
	"fmt"

	"cse-go/internal/commandbus" // [已更新] 导入新的共享接口包
	pb "cse-go/pkg/api/v1"
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
		Description:      "获取所有可用的打印机列表 (在当前操作系统不受支持)。",
		ParametersSchema: `{}`,
		ResultSchema:     `{"type": "array", "items": {"type": "string"}}`,
	}
}

// Execute 在非 Windows 系统上返回一个空列表
func (c *GetPrintersCmd) Execute(params *pb.CommandParams) (*pb.CommandResult, error) {
	fmt.Println("Warning: 'print.getPrinters' is not supported on this OS.")
	emptyList, _ := json.Marshal([]string{})
	return &pb.CommandResult{
		JsonPayload: string(emptyList),
	}, nil
}
