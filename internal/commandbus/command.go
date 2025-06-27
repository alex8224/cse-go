package commandbus

import (
	pb "cse-go/pkg/api/v1" // 导入 gRPC API
)

// Command 是所有具体命令处理器必须实现的通用接口
type Command interface {
	// Name 返回命令的唯一名称，例如 "print.getPrinters"
	Name() string

	// Execute 执行命令逻辑
	// 它接收 gRPC 请求中的参数，并返回一个 gRPC 响应结果
	Execute(params *pb.CommandParams) (*pb.CommandResult, error)

	// GetInfo 返回命令的元数据，用于服务发现
	GetInfo() *pb.CommandInfo
}
