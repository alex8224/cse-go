package http

import "net/http"

func (s *Server) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// API V1 路由组
	mux.HandleFunc("/api/v1/components", s.listComponentsHandler())
	mux.HandleFunc("/api/v1/execute", s.executeCommandHandler())

	// 未来可以添加 /api/v2/... 等

	return mux
}
