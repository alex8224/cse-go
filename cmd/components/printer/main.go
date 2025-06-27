package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"cse-go/cmd/components/printer/commands"

	"cse-go/internal/commandbus"
	pb "cse-go/pkg/api/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// printerServer 实现了 ComponentService
type printerServer struct {
	pb.UnimplementedComponentServiceServer
	grpcServer *grpc.Server
	// [已更新] map 的 value 类型现在是新的共享接口
	commandMap map[string]commandbus.Command
}

// NewPrinterServer 创建一个新的 printerServer 实例并注册所有命令
func NewPrinterServer(grpcServer *grpc.Server) *printerServer {
	s := &printerServer{
		grpcServer: grpcServer,
		commandMap: make(map[string]commandbus.Command),
	}
	s.registerCommands()
	return s
}

// registerCommands 初始化并注册所有支持的命令
func (s *printerServer) registerCommands() {
	// [自动注册] 从全局注册器获取所有已注册的命令
	autoCommands := commands.GlobalRegistry.GetCommands()
	for name, cmd := range autoCommands {
		s.commandMap[name] = cmd
		log.Printf("[Printer Component] 命令 '%s' 已自动加载到组件中", name)
	}

	log.Printf("[Printer Component] 共加载了 %d 个命令", len(s.commandMap))
}

// ExecuteCommand 从注册表中查找并执行命令
func (s *printerServer) ExecuteCommand(ctx context.Context, req *pb.ExecuteCommandRequest) (*pb.ExecuteCommandResponse, error) {
	commandName := req.GetCommandName()
	log.Printf("[Printer Component] 收到命令执行请求: '%s'", commandName)

	cmd, ok := s.commandMap[commandName]
	if !ok {
		errMsg := fmt.Sprintf("命令 '%s' 未找到或不受支持。", commandName)
		return &pb.ExecuteCommandResponse{Success: false, ErrorMessage: errMsg}, nil
	}

	result, err := cmd.Execute(req.GetParams())
	if err != nil {
		return &pb.ExecuteCommandResponse{Success: false, ErrorMessage: err.Error()}, nil
	}

	return &pb.ExecuteCommandResponse{Success: true, Result: result}, nil
}

// GetMetadata 从注册表中动态生成命令列表
func (s *printerServer) GetMetadata(ctx context.Context, req *pb.GetMetadataRequest) (*pb.ComponentMetadata, error) {
	log.Println("[Printer Component] Supervisor 调用了 GetMetadata 方法")

	providedCmds := make([]*pb.CommandInfo, 0, len(s.commandMap))
	for _, cmd := range s.commandMap {
		providedCmds = append(providedCmds, cmd.GetInfo())
	}

	return &pb.ComponentMetadata{
		Name:             "printer",
		Version:          "1.2.0-shared-interface",
		Description:      "一个支持跨平台命令的打印组件。",
		Author:           "CSE Team",
		ProvidedCommands: providedCmds,
	}, nil
}

// ... GetStatus, Shutdown, startMyService, registerToSupervisor, 和 main 函数保持不变 ...

func (s *printerServer) GetStatus(ctx context.Context, req *pb.GetStatusRequest) (*pb.GetStatusResponse, error) {
	return &pb.GetStatusResponse{CurrentState: pb.ComponentState_RUNNING, Message: "打印组件正在运行"}, nil
}

func (s *printerServer) Shutdown(ctx context.Context, req *pb.ShutdownRequest) (*pb.ShutdownResponse, error) {
	log.Println("[Printer Component] Supervisor 调用了 Shutdown 方法...")
	
	// 先返回响应，确保supervisor收到确认
	response := &pb.ShutdownResponse{Acknowledged: true, Message: "Shutdown request received."}
	
	// 延迟关闭，给响应足够时间发送
	go func() {
		time.Sleep(100 * time.Millisecond) // 缩短延迟，确保响应能发送
		log.Println("[Printer Component] 正在优雅关闭gRPC服务器...")
		s.grpcServer.GracefulStop()
		log.Println("[Printer Component] 组件已关闭")
		os.Exit(0)
	}()
	
	return response, nil
}

func startMyService() (net.Listener, *grpc.Server) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("无法监听动态端口: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterComponentServiceServer(s, NewPrinterServer(s))
	log.Printf("打印组件的服务启动，正在动态监听 %s", lis.Addr().String())
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("gRPC 服务已停止: %v", err)
		}
	}()
	return lis, s
}

func registerToSupervisor(discoveryAddr, myAddr, componentName string) {
	conn, err := grpc.NewClient(discoveryAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("无法连接到发现服务 at %s: %v", discoveryAddr, err)
	}
	defer conn.Close()
	client := pb.NewComponentDiscoveryServiceClient(conn)
	req := &pb.RegisterComponentRequest{
		Name: componentName, GrpcAddress: myAddr, Pid: int32(os.Getpid()),
	}
	log.Printf("正在向发现服务 (%s) 注册...", discoveryAddr)
	res, err := client.RegisterComponent(context.Background(), req)
	if err != nil {
		log.Fatalf("注册失败: %v", err)
	}
	if res.Success {
		log.Printf("成功注册到 Supervisor: %s", res.Message)
	} else {
		log.Fatalf("注册被 Supervisor 拒绝: %s", res.Message)
	}
}

func main() {
	discoveryAddr := flag.String("discovery-addr", "", "Supervisor's discovery service address")
	componentName := flag.String("component-name", "", "This component's name")
	flag.Parse()
	if *discoveryAddr == "" || *componentName == "" {
		log.Fatal("必须提供 --discovery-addr 和 --component-name 参数")
	}
	listener, _ := startMyService()
	myAddress := listener.Addr().String()
	registerToSupervisor(*discoveryAddr, myAddress, *componentName)
	select {}
}
