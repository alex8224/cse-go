// Package commands 包含了打印组件特定的命令实现
package commands

import (
	"encoding/json"
	"log"

	"cse-go/internal/commandbus"
	pb "cse-go/pkg/api/v1"
	winprinter "github.com/godoes/printers"
)

// 确保 SetDefaultPrinterCmd 实现了 commandbus.Command 接口
var _ commandbus.Command = (*SetDefaultPrinterCmd)(nil)

// SetDefaultPrinterCmd 结构体定义
type SetDefaultPrinterCmd struct{}

// Name 返回命令名称
func (c *SetDefaultPrinterCmd) Name() string {
	return "print.setDefaultPrinter"
}

// GetInfo 返回命令元数据
func (c *SetDefaultPrinterCmd) GetInfo() *pb.CommandInfo {
	return &pb.CommandInfo{
		CommandName:      c.Name(),
		Description:      "设置系统默认打印机 (Windows)。",
		ParametersSchema: `{"type": "object", "properties": {"printerName": {"type": "string", "description": "要设置为默认的打印机名称"}}, "required": ["printerName"]}`,
		ResultSchema:     `{"type": "object", "properties": {"success": {"type": "boolean"}, "printerName": {"type": "string"}, "message": {"type": "string"}}}`,
	}
}

// Execute 设置系统默认打印机
func (c *SetDefaultPrinterCmd) Execute(params *pb.CommandParams) (*pb.CommandResult, error) {
	// 解析参数
	var requestParams struct {
		PrinterName string `json:"printerName"`
	}

	if err := json.Unmarshal([]byte(params.JsonPayload), &requestParams); err != nil {
		log.Printf("解析参数失败: %v", err)
		result := map[string]interface{}{
			"success":     false,
			"printerName": "",
			"message":     "参数解析失败: " + err.Error(),
		}
		resultJson, _ := json.Marshal(result)
		return &pb.CommandResult{
			JsonPayload: string(resultJson),
		}, nil
	}

	// 验证打印机名称参数
	if requestParams.PrinterName == "" {
		log.Printf("打印机名称不能为空")
		result := map[string]interface{}{
			"success":     false,
			"printerName": "",
			"message":     "打印机名称不能为空",
		}
		resultJson, _ := json.Marshal(result)
		return &pb.CommandResult{
			JsonPayload: string(resultJson),
		}, nil
	}

	// 首先验证打印机是否存在
	printerList, err := winprinter.ReadNames()
	if err != nil {
		log.Printf("获取打印机列表失败: %v", err)
		result := map[string]interface{}{
			"success":     false,
			"printerName": requestParams.PrinterName,
			"message":     "无法验证打印机是否存在: " + err.Error(),
		}
		resultJson, _ := json.Marshal(result)
		return &pb.CommandResult{
			JsonPayload: string(resultJson),
		}, nil
	}

	// 检查指定的打印机是否存在
	printerExists := false
	for _, printer := range printerList {
		if printer == requestParams.PrinterName {
			printerExists = true
			break
		}
	}

	if !printerExists {
		log.Printf("指定的打印机不存在: %s", requestParams.PrinterName)
		result := map[string]interface{}{
			"success":     false,
			"printerName": requestParams.PrinterName,
			"message":     "指定的打印机不存在: " + requestParams.PrinterName,
		}
		resultJson, _ := json.Marshal(result)
		return &pb.CommandResult{
			JsonPayload: string(resultJson),
		}, nil
	}

	// 设置默认打印机
	err = winprinter.SetDefault(requestParams.PrinterName)
	if err != nil {
		log.Printf("设置默认打印机失败: %v", err)
		result := map[string]interface{}{
			"success":     false,
			"printerName": requestParams.PrinterName,
			"message":     "设置默认打印机失败: " + err.Error(),
		}
		resultJson, _ := json.Marshal(result)
		return &pb.CommandResult{
			JsonPayload: string(resultJson),
		}, nil
	}

	log.Printf("成功设置默认打印机: %s", requestParams.PrinterName)
	result := map[string]interface{}{
		"success":     true,
		"printerName": requestParams.PrinterName,
		"message":     "成功设置默认打印机",
	}
	resultJson, _ := json.Marshal(result)
	return &pb.CommandResult{
		JsonPayload: string(resultJson),
	}, nil
}

// init 自动注册命令
func init() {
	GlobalRegistry.Register(&SetDefaultPrinterCmd{})
}