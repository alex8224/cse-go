package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	pb "cse-go/pkg/api/v1" // 导入 gRPC API

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// 定义主程序发现服务的固定地址
	discoveryServiceAddress = "localhost:50050"
)

// ComponentConfig 定义了组件配置文件的结构 (移除了 grpc_address)
type ComponentConfig struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Cmd         string   `json:"cmd"`
	CmdArgs     []string `json:"cmd_args"`
}

// ComponentInfo 存储了一个已注册组件的信息和 gRPC 客户端
type ComponentInfo struct {
	Config   *ComponentConfig
	Metadata *pb.ComponentMetadata
	Client   pb.ComponentServiceClient
	Cmd      *exec.Cmd
}

// ComponentManager 负责管理所有组件的生命周期
type ComponentManager struct {
	components map[string]*ComponentInfo
	lock       sync.RWMutex
}

// NewComponentManager 创建一个新的组件管理器
func NewComponentManager() *ComponentManager {
	return &ComponentManager{
		components: make(map[string]*ComponentInfo),
	}
}

// discoveryServer 实现了新的 ComponentDiscoveryService
type discoveryServer struct {
	pb.UnimplementedComponentDiscoveryServiceServer
	manager *ComponentManager // 引用组件管理器
}

// RegisterComponent 是组件用来注册自己的方法
func (s *discoveryServer) RegisterComponent(ctx context.Context, req *pb.RegisterComponentRequest) (*pb.RegisterComponentResponse, error) {
	log.Printf("[Discovery Service] 收到来自组件 '%s' (PID: %d) 的注册请求，地址: %s", req.Name, req.Pid, req.GrpcAddress)

	// 连接到组件报告的地址，以获取元数据
	conn, err := grpc.Dial(req.GrpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("[Discovery Service] 错误: 无法连接回组件 '%s': %v", req.Name, err)
		return &pb.RegisterComponentResponse{Success: false, Message: "Failed to connect back to component"}, err
	}
	// 注意: 实际应用中需要管理这些连接的生命周期
	// defer conn.Close()

	client := pb.NewComponentServiceClient(conn)
	metadata, err := client.GetMetadata(context.Background(), &pb.GetMetadataRequest{})
	if err != nil {
		log.Printf("[Discovery Service] 错误: 无法从组件 '%s' 获取元数据: %v", req.Name, err)
		return &pb.RegisterComponentResponse{Success: false, Message: "Failed to get metadata"}, err
	}

	// 将组件信息存储到管理器中
	s.manager.lock.Lock()
	// 注意: 这里我们假设 ComponentInfo 里的 Cmd 和 Config 稍后会被填充
	// 一个更完整的实现会使用一种方式将这里的注册信息和启动时的进程信息关联起来
	// 为了简化，我们暂时只存储关键信息
	if compInfo, ok := s.manager.components[req.Name]; ok {
		compInfo.Metadata = metadata
		compInfo.Client = client
	} else {
		// 这是个问题，意味着组件被注册时，我们还没有它的启动信息
		// 我们将在 "LaunchComponents" 中先创建一个占位符
		log.Printf("警告: 组件 '%s' 注册时未找到占位符信息", req.Name)
	}
	s.manager.lock.Unlock()

	log.Printf("[Discovery Service] 组件 '%s' v%s 注册成功！", metadata.Name, metadata.Version)

	return &pb.RegisterComponentResponse{Success: true, Message: "Registration successful"}, nil
}

// startDiscoveryService 启动监听组件注册的服务
func startDiscoveryService(manager *ComponentManager) {
	lis, err := net.Listen("tcp", discoveryServiceAddress)
	if err != nil {
		log.Fatalf("无法监听发现服务端口 %s: %v", discoveryServiceAddress, err)
	}
	s := grpc.NewServer()
	pb.RegisterComponentDiscoveryServiceServer(s, &discoveryServer{manager: manager})

	log.Printf("组件发现服务启动成功，正在监听 %s", discoveryServiceAddress)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("启动发现服务失败: %v", err)
	}
}

// LaunchComponents 扫描配置目录，仅负责启动组件进程
func (m *ComponentManager) LaunchComponents(configDir string) {
	files, err := ioutil.ReadDir(configDir)
	if err != nil {
		log.Fatalf("无法读取配置目录 '%s': %v", configDir, err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			configPath := filepath.Join(configDir, file.Name())
			// ... 读取和解析配置文件 ...
			configData, err := ioutil.ReadFile(configPath)
			if err != nil {
				log.Printf("错误: 无法读取配置文件 %s: %v", configPath, err)
				continue
			}
			// 获取当前可执行文件的绝对路径
			exePath, err := os.Executable()
			if err != nil {
				log.Printf("错误: 无法获取当前可执行文件路径: %v", err)
				continue
			}
			// 获取可执行文件所在目录
			exeDir := filepath.Dir(exePath)
			log.Printf("当前可执行文件路径: %s", exeDir)
			// 将相对路径转换为绝对路径
			configPath = filepath.Join(exeDir, configPath)
			var config ComponentConfig
			if err := json.Unmarshal(configData, &config); err != nil {
				log.Printf("错误: 解析配置文件 %s 失败: %v", configPath, err)
				continue
			}

			// 将发现服务的地址作为命令行参数传递给组件
			args := append(config.CmdArgs, "--discovery-addr="+discoveryServiceAddress, "--component-name="+config.Name)
			cmd := exec.Command(filepath.Join(exeDir, config.Cmd), args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			// 在组件启动前，先在 map 中创建一个占位符
			m.lock.Lock()
			m.components[config.Name] = &ComponentInfo{
				Config: &config,
				Cmd:    cmd,
			}
			m.lock.Unlock()

			if err := cmd.Start(); err != nil {
				log.Printf("错误: 启动组件 '%s' 失败: %v", config.Name, err)
				m.lock.Lock()
				delete(m.components, config.Name) // 启动失败则删除占位符
				m.lock.Unlock()
				continue
			}
			log.Printf("组件 '%s' 进程已启动 (PID: %d)，等待其主动注册...", config.Name, cmd.Process.Pid)
			fmt.Println("---")
		}
	}
}

func main() {
	log.Println("CSE 主应用程序 (Supervisor) 启动...")
	manager := NewComponentManager()

	// 在一个单独的 goroutine 中启动发现服务
	go startDiscoveryService(manager)

	// 等待一秒，确保发现服务完全启动
	time.Sleep(time.Second)

	// 主 goroutine 负责启动所有组件
	manager.LaunchComponents("./configs")

	log.Println("所有组件已启动。主程序正在运行...")

	// 保持主程序运行
	select {}
}
