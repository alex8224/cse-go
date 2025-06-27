package http

import (
	"log"
	"net/http"

	"cse-go/cmd/supervisor/manager"
)

// Server 是我们的 HTTP 服务器结构体
type Server struct {
	addr    string
	manager *manager.ComponentManager
}

// NewServer 创建一个新的 HTTP 服务器实例
func NewServer(addr string, manager *manager.ComponentManager) *Server {
	return &Server{
		addr:    addr,
		manager: manager,
	}
}

// Start 启动 HTTP 服务器并开始监听
func (s *Server) Start() {
	log.Printf("HTTP 服务启动，正在监听 %s", s.addr)

	// 设置路由
	mux := s.setupRoutes()

	// 启动服务器
	if err := http.ListenAndServe(s.addr, mux); err != nil {
		log.Fatalf("启动 HTTP 服务失败: %v", err)
	}
}
