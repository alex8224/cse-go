// Package commands 包含了打印组件特定的命令实现
package commands

import (
	"cse-go/internal/commandbus"
	"log"
	"sync"
)

// CommandRegistry 命令注册器
type CommandRegistry struct {
	commands map[string]commandbus.Command
	mu       sync.RWMutex
}

// NewCommandRegistry 创建新的命令注册器
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]commandbus.Command),
	}
}

// Register 注册命令
func (r *CommandRegistry) Register(cmd commandbus.Command) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	cmdName := cmd.Name()
	if _, exists := r.commands[cmdName]; exists {
		log.Printf("[Auto Registry] 警告: 命令 '%s' 已存在，将被覆盖", cmdName)
	}
	
	r.commands[cmdName] = cmd
	log.Printf("[Auto Registry] 命令 '%s' 已自动注册", cmdName)
}

// GetCommands 获取所有注册的命令
func (r *CommandRegistry) GetCommands() map[string]commandbus.Command {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// 返回副本以避免并发修改
	result := make(map[string]commandbus.Command)
	for name, cmd := range r.commands {
		result[name] = cmd
	}
	return result
}

// GetCommandCount 获取已注册命令的数量
func (r *CommandRegistry) GetCommandCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.commands)
}

// ListCommands 列出所有已注册的命令名称
func (r *CommandRegistry) ListCommands() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	return names
}

// 全局注册器实例
var GlobalRegistry = NewCommandRegistry()