package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	pb "cse-go/pkg/api/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ComponentConfig 定义了组件配置文件的结构。
type ComponentConfig struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Cmd         string   `json:"cmd"`
	CmdArgs     []string `json:"cmd_args"`
}

// ComponentInfo 存储了一个已注册组件的完整信息。
type ComponentInfo struct {
	Config   *ComponentConfig
	Metadata *pb.ComponentMetadata
	Client   pb.ComponentServiceClient
	Cmd      *exec.Cmd
	Conn     *grpc.ClientConn // gRPC connection to the component
}

// ComponentManager 负责管理所有组件的生命周期。
type ComponentManager struct {
	Components map[string]*ComponentInfo
	lock       sync.RWMutex
}

// NewComponentManager 创建一个新的组件管理器。
func NewComponentManager() *ComponentManager {
	return &ComponentManager{
		Components: make(map[string]*ComponentInfo),
	}
}

// LaunchComponents 扫描配置目录，并启动所有组件进程。
func (m *ComponentManager) LaunchComponents(configDir, discoveryAddr string) {
	files, err := os.ReadDir(configDir)
	if err != nil {
		log.Fatalf("无法读取配置目录 '%s': %v", configDir, err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		configPath := filepath.Join(configDir, file.Name())
		configData, err := os.ReadFile(configPath)
		if err != nil {
			log.Printf("错误: 无法读取配置文件 %s: %v", configPath, err)
			continue
		}

		var config ComponentConfig
		if err := json.Unmarshal(configData, &config); err != nil {
			log.Printf("错误: 解析配置文件 %s 失败: %v", configPath, err)
			continue
		}

		// 将发现服务的地址作为命令行参数传递给组件
		args := append(config.CmdArgs, "--discovery-addr="+discoveryAddr, "--component-name="+config.Name)

		// 获取主程序所在目录，以正确地定位组件可执行文件
		exePath, err := os.Executable()
		if err != nil {
			log.Printf("错误: 无法获取主程序路径: %v", err)
			continue
		}
		exeDir := filepath.Dir(exePath)

		cmd := exec.Command(filepath.Join(exeDir, config.Cmd), args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// 在组件启动前，先在 map 中创建一个占位符
		m.lock.Lock()
		m.Components[config.Name] = &ComponentInfo{
			Config: &config,
			Cmd:    cmd,
		}
		m.lock.Unlock()

		if err := cmd.Start(); err != nil {
			log.Printf("错误: 启动组件 '%s' 失败: %v", config.Name, err)
			m.lock.Lock()
			delete(m.Components, config.Name) // 启动失败则删除占位符
			m.lock.Unlock()
			continue
		}
		log.Printf("组件 '%s' 进程已启动 (PID: %d)，等待其主动注册...", config.Name, cmd.Process.Pid)
	}
}

// HandleRegistration 处理来自组件的注册请求。
func (m *ComponentManager) HandleRegistration(req *pb.RegisterComponentRequest) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	compInfo, ok := m.Components[req.Name]
	if !ok {
		return fmt.Errorf("收到未知的组件注册请求: %s", req.Name)
	}

	// 连接到组件报告的地址，以获取元数据
	conn, err := grpc.Dial(req.GrpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("无法连接回组件 '%s': %w", req.Name, err)
	}

	client := pb.NewComponentServiceClient(conn)
	metadata, err := client.GetMetadata(context.Background(), &pb.GetMetadataRequest{})
	if err != nil {
		conn.Close()
		return fmt.Errorf("无法从组件 '%s' 获取元数据: %w", req.Name, err)
	}

	// 更新组件信息
	compInfo.Metadata = metadata
	compInfo.Client = client
	compInfo.Conn = conn

	log.Printf("[Discovery Service] 组件 '%s' v%s 注册成功！", metadata.Name, metadata.Version)
	return nil
}

// ShutdownAllComponents 优雅地关闭所有已注册的组件。
func (m *ComponentManager) ShutdownAllComponents() {
	m.lock.RLock()
	defer m.lock.RUnlock()

	log.Println("正在向所有组件发送关闭信号...")
	var wg sync.WaitGroup
	for name, comp := range m.Components {
		if comp.Client == nil {
			continue // 组件未成功注册
		}
		wg.Add(1)
		go func(name string, comp *ComponentInfo) {
			defer wg.Done()
			log.Printf("正在关闭组件: %s...", name)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, err := comp.Client.Shutdown(ctx, &pb.ShutdownRequest{})
			if err != nil {
				log.Printf("关闭组件 '%s' 时出错 (可能已被终止): %v", name, err)
				// 如果 gRPC 调用失败，直接终止进程
				comp.Cmd.Process.Kill()
			}
			// 关闭 gRPC 连接
			if comp.Conn != nil {
				comp.Conn.Close()
			}
		}(name, comp)
	}
	wg.Wait()
	log.Println("所有组件已关闭。")
}

// Lock 提供对互斥锁的写锁定
func (m *ComponentManager) Lock() {
	m.lock.Lock()
}

// Unlock 提供对互斥锁的写解锁
func (m *ComponentManager) Unlock() {
	m.lock.Unlock()
}

// RLock 提供对互斥锁的读锁定
func (m *ComponentManager) RLock() {
	m.lock.RLock()
}

// RUnlock 提供对互斥锁的读解锁
func (m *ComponentManager) RUnlock() {
	m.lock.RUnlock()
}
