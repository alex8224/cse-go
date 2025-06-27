//go:build windows

// Package commands 包含了打印组件特定的命令实现
package commands

import (
	"encoding/json"
	"log"

	"cse-go/internal/commandbus"
	pb "cse-go/pkg/api/v1"
	winprinter "github.com/godoes/printers"
)

// 确保 GetPrintersCmd 实现了 commandbus.Command 接口
var _ commandbus.Command = (*GetDefaultPrinterCmd)(nil)

// GetDefaultPrinterCmd 结构体定义
type GetDefaultPrinterCmd struct{}

// Name 返回命令名称
func (c *GetDefaultPrinterCmd) Name() string {
	return "print.getDefaultPrinter"
}

// GetInfo 返回命令元数据
func (c *GetDefaultPrinterCmd) GetInfo() *pb.CommandInfo {
	return &pb.CommandInfo{
		CommandName:      c.Name(),
		Description:      "获取系统默认打印机 (Windows)。",
		ParametersSchema: `{}`,
		ResultSchema:     `{"type": "object", "properties": {"success": {"type": "boolean"}, "defaultPrinter": {"type": "string"}, "message": {"type": "string"}}}`,
	}
}

// Execute 获取系统默认打印机
func (c *GetDefaultPrinterCmd) Execute(params *pb.CommandParams) (*pb.CommandResult, error) {
	// 获取默认打印机
	defaultPrinter, err := winprinter.GetDefault()
	if err != nil {
		log.Printf("获取默认打印机失败: %v", err)
		result := map[string]interface{}{
			"success":        false,
			"defaultPrinter": "",
			"message":        "获取默认打印机失败: " + err.Error(),
		}
		resultJson, _ := json.Marshal(result)
		return &pb.CommandResult{
			JsonPayload: string(resultJson),
		}, nil
	}

	if defaultPrinter == "" {
		log.Printf("系统未设置默认打印机")
		result := map[string]interface{}{
			"success":        false,
			"defaultPrinter": "",
			"message":        "系统未设置默认打印机",
		}
		resultJson, _ := json.Marshal(result)
		return &pb.CommandResult{
			JsonPayload: string(resultJson),
		}, nil
	}

	log.Printf("成功获取默认打印机: %s", defaultPrinter)
	result := map[string]interface{}{
		"success":        true,
		"defaultPrinter": defaultPrinter,
		"message":        "成功获取默认打印机",
	}
	resultJson, _ := json.Marshal(result)
	return &pb.CommandResult{
		JsonPayload: string(resultJson),
	}, nil
}

// init 自动注册命令
func init() {
	GlobalRegistry.Register(&GetDefaultPrinterCmd{})
}
