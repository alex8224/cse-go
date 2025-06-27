package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cse-go/cmd/supervisor/http"
	"cse-go/cmd/supervisor/manager" // [已更新] 导入新的 manager 包
	pb "cse-go/pkg/api/v1"

	utils "cse-go/cmd/utils"

	"google.golang.org/grpc"
)

const (
	discoveryServiceAddress = "localhost:50050"
	httpServiceAddress      = "localhost:18848"
)

// discoveryServer 实现了 ComponentDiscoveryService
type discoveryServer struct {
	pb.UnimplementedComponentDiscoveryServiceServer
	manager *manager.ComponentManager
}

// RegisterComponent 将注册逻辑委托给 ComponentManager
func (s *discoveryServer) RegisterComponent(ctx context.Context, req *pb.RegisterComponentRequest) (*pb.RegisterComponentResponse, error) {
	err := s.manager.HandleRegistration(req)
	if err != nil {
		log.Printf("[Discovery Service] 错误: 处理组件 '%s' 注册失败: %v", req.Name, err)
		return &pb.RegisterComponentResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.RegisterComponentResponse{Success: true, Message: ""}, nil
}

// startDiscoveryService 启动监听组件注册的 gRPC 服务
func startDiscoveryService(manager *manager.ComponentManager) *grpc.Server {
	lis, err := net.Listen("tcp", discoveryServiceAddress)
	if err != nil {
		log.Fatalf("无法监听发现服务端口 %s: %v", discoveryServiceAddress, err)
	}

	s := grpc.NewServer()
	pb.RegisterComponentDiscoveryServiceServer(s, &discoveryServer{manager: manager})

	go func() {
		log.Printf("组件发现服务启动成功，正在监听 %s", discoveryServiceAddress)
		if err := s.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Fatalf("发现服务意外停止: %v", err)
		}
	}()
	return s
}

func main() {
	log.Printf("CSE 主应用程序 (Supervisor) 启动, 当前操作系统 %s...", utils.GetOSType())
	compManager := manager.NewComponentManager()

	// 1. 启动 gRPC 发现服务
	discoveryGrpcServer := startDiscoveryService(compManager)

	// 2. 启动 HTTP API 服务
	httpServer := http.NewServer(httpServiceAddress, compManager)
	go httpServer.Start()

	// 等待服务启动
	time.Sleep(time.Second)

	// 3. 启动所有组件
	compManager.LaunchComponents("./configs", discoveryServiceAddress)

	log.Println("所有服务已启动。按 Ctrl+C 关闭。")

	// 4. 等待关闭信号以实现优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("收到关闭信号，正在关闭所有服务...")

	// 优雅地关闭所有组件
	compManager.ShutdownAllComponents()

	// 优雅地关闭 gRPC 服务
	discoveryGrpcServer.GracefulStop()

	// 注意: HTTP 服务器的优雅关闭可以在 http/server.go 中实现，
	// 此处为简化暂未添加，但实际生产中应添加。

	log.Println("所有服务已成功关闭。")
}
