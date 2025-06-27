//go:build windows

// Package commands 包含了打印组件特定的命令实现
package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cse-go/internal/commandbus"
	pb "cse-go/pkg/api/v1"

	winprinter "github.com/godoes/printers"
)

// 确保 GetPrintersCmd 实现了 commandbus.Command 接口
var _ commandbus.Command = (*PrintTestCmd)(nil)

// PrintTestCmd 结构体定义
type PrintTestCmd struct{}

// Name 返回命令名称
func (c *PrintTestCmd) Name() string {
	return "print.testPrint"
}

// GetInfo 返回命令元数据
func (c *PrintTestCmd) GetInfo() *pb.CommandInfo {
	return &pb.CommandInfo{
		CommandName:      c.Name(),
		Description:      "发送测试页到指定的打印机 (Windows)。",
		ParametersSchema: `{"type": "object", "properties": {"printerName": {"type": "string", "description": "打印机名称"}}, "required": ["printerName"]}`,
		ResultSchema:     `{"type": "object", "properties": {"success": {"type": "boolean"}, "message": {"type": "string"}}}`,
	}
}

// Execute 发送测试页到指定的打印机
func (c *PrintTestCmd) Execute(params *pb.CommandParams) (*pb.CommandResult, error) {
	// 解析参数
	var requestParams struct {
		PrinterName string `json:"printerName"`
	}

	if err := json.Unmarshal([]byte(params.JsonPayload), &requestParams); err != nil {
		log.Printf("解析参数失败: %v", err)
		result := map[string]interface{}{
			"success": false,
			"message": "参数解析失败: " + err.Error(),
		}
		resultJson, _ := json.Marshal(result)
		return &pb.CommandResult{
			JsonPayload: string(resultJson),
		}, nil
	}

	if requestParams.PrinterName == "" {
		result := map[string]interface{}{
			"success": false,
			"message": "打印机名称不能为空",
		}
		resultJson, _ := json.Marshal(result)
		return &pb.CommandResult{
			JsonPayload: string(resultJson),
		}, nil
	}

	// 打开打印机
	printer, err := winprinter.Open(requestParams.PrinterName)
	if err != nil {
		log.Printf("打开打印机失败: %v", err)
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("无法打开打印机 '%s': %v", requestParams.PrinterName, err),
		}
		resultJson, _ := json.Marshal(result)
		return &pb.CommandResult{
			JsonPayload: string(resultJson),
		}, nil
	}
	defer printer.Close()

	// 开始打印文档
	err = printer.StartDocument("测试页", "RAW")
	if err != nil {
		log.Printf("开始打印文档失败: %v", err)
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("开始打印文档失败: %v", err),
		}
		resultJson, _ := json.Marshal(result)
		return &pb.CommandResult{
			JsonPayload: string(resultJson),
		}, nil
	}
	defer printer.EndDocument()

	// 开始页面
	err = printer.StartPage()
	if err != nil {
		log.Printf("开始页面失败: %v", err)
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("开始页面失败: %v", err),
		}
		resultJson, _ := json.Marshal(result)
		return &pb.CommandResult{
			JsonPayload: string(resultJson),
		}, nil
	}
	defer printer.EndPage()

	// 写入测试内容
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	testContent := fmt.Sprintf(`打印机测试页

打印机名称: %s
打印时间: %s

这是一个测试页面，用于验证打印机是否正常工作。
如果您能看到这个页面，说明打印机工作正常。

功能测试项目:
✓ 打印机连接正常
✓ 数据传输正常
✓ 文本输出正常

测试完成。`, requestParams.PrinterName, currentTime)

	_, err = printer.Write([]byte(testContent))
	if err != nil {
		log.Printf("写入测试内容失败: %v", err)
		result := map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("写入测试内容失败: %v", err),
		}
		resultJson, _ := json.Marshal(result)
		return &pb.CommandResult{
			JsonPayload: string(resultJson),
		}, nil
	}

	log.Printf("成功发送测试页到打印机: %s", requestParams.PrinterName)
	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("测试页已成功发送到打印机 '%s'", requestParams.PrinterName),
	}
	resultJson, _ := json.Marshal(result)
	return &pb.CommandResult{
		JsonPayload: string(resultJson),
	}, nil
}

// init 自动注册命令
func init() {
	GlobalRegistry.Register(&PrintTestCmd{})
}
