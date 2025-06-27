package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	pb "cse-go/pkg/api/v1"
)

// --- DTO (Data Transfer Objects) ---

// listComponentsResponse 定义了 /api/v1/components 的响应结构
type listComponentsResponse struct {
	Components []componentInfo `json:"components"`
}

type componentInfo struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Description      string            `json:"description"`
	ProvidedCommands []*pb.CommandInfo `json:"provided_commands"`
}

// executeRequest 定义了 /api/v1/execute 的请求体结构
type executeRequest struct {
	ComponentName string `json:"component_name"`
	CommandName   string `json:"command_name"`
	Params        any    `json:"params"` // 使用 any 接收任意 JSON 结构
}

// --- Handlers ---

// listComponentsHandler 返回一个处理器，用于列出所有已注册的组件
func (s *Server) listComponentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		s.manager.Lock()
		defer s.manager.Unlock()

		resp := listComponentsResponse{
			Components: make([]componentInfo, 0, len(s.manager.Components)),
		}

		for _, comp := range s.manager.Components {
			if comp.Metadata != nil { // 确保组件已成功注册并返回了元数据
				resp.Components = append(resp.Components, componentInfo{
					Name:             comp.Metadata.Name,
					Version:          comp.Metadata.Version,
					Description:      comp.Metadata.Description,
					ProvidedCommands: comp.Metadata.ProvidedCommands,
				})
			}
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

// executeCommandHandler 返回一个处理器，用于执行组件命令
func (s *Server) executeCommandHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 解析请求体
		var req executeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// 查找组件
		s.manager.RLock()
		comp, ok := s.manager.Components[req.ComponentName]
		s.manager.RUnlock()

		if !ok || comp.Client == nil {
			http.Error(w, "Component not found or not ready", http.StatusNotFound)
			return
		}

		// 序列化参数
		paramsPayload, err := json.Marshal(req.Params)
		if err != nil {
			http.Error(w, "Invalid params format", http.StatusBadRequest)
			return
		}

		// 准备 gRPC 请求
		grpcReq := &pb.ExecuteCommandRequest{
			CommandName: req.CommandName,
			Params: &pb.CommandParams{
				JsonPayload: string(paramsPayload),
			},
		}

		// 执行 gRPC 调用
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		log.Printf("Executing command '%s' on component '%s'", req.CommandName, req.ComponentName)
		grpcResp, err := comp.Client.ExecuteCommand(ctx, grpcReq)
		if err != nil {
			log.Printf("gRPC call failed: %v", err)
			http.Error(w, "Failed to execute command: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 处理 gRPC 响应
		if !grpcResp.Success {
			writeJSON(w, http.StatusOK, map[string]any{
				"success": false,
				"error":   grpcResp.ErrorMessage,
			})
			return
		}

		// 反序列化结果并返回
		var resultData any
		if err := json.Unmarshal([]byte(grpcResp.GetResult().GetJsonPayload()), &resultData); err != nil {
			http.Error(w, "Failed to parse command result", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"data":    resultData,
		})
	}
}

// writeJSON 是一个辅助函数，用于统一写入 JSON 响应
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
}
