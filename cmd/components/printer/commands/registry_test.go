package commands

import (
	"testing"
)

// TestAutoRegistration 测试自动注册功能
func TestAutoRegistration(t *testing.T) {
	// 获取所有已注册的命令
	commands := GlobalRegistry.GetCommands()
	
	// 验证预期的命令都已注册
	expectedCommands := []string{
		"print.getPrinters",
		"print.getDefaultPrinter",
		"print.setDefaultPrinter",
		"print.testPrint",
	}
	
	for _, expectedCmd := range expectedCommands {
		if _, exists := commands[expectedCmd]; !exists {
			t.Errorf("命令 '%s' 未被自动注册", expectedCmd)
		}
	}
	
	// 验证注册的命令数量
	if len(commands) != len(expectedCommands) {
		t.Errorf("预期注册 %d 个命令，实际注册了 %d 个命令", len(expectedCommands), len(commands))
	}
	
	// 验证每个命令都能正确返回名称
	for cmdName, cmd := range commands {
		if cmd.Name() != cmdName {
			t.Errorf("命令名称不匹配: 注册名称='%s', 命令返回名称='%s'", cmdName, cmd.Name())
		}
	}
	
	t.Logf("自动注册测试通过，成功注册了 %d 个命令", len(commands))
}

// TestRegistryThreadSafety 测试注册器的线程安全性
func TestRegistryThreadSafety(t *testing.T) {
	// 创建新的注册器用于测试
	testRegistry := NewCommandRegistry()
	
	// 并发注册命令
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			testRegistry.Register(&GetPrintersCmd{})
			done <- true
		}()
	}
	
	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// 验证只注册了一个命令（重复注册应该被覆盖）
	commands := testRegistry.GetCommands()
	if len(commands) != 1 {
		t.Errorf("预期注册 1 个命令，实际注册了 %d 个命令", len(commands))
	}
	
	if _, exists := commands["print.getPrinters"]; !exists {
		t.Error("命令 'print.getPrinters' 未被注册")
	}
	
	t.Log("线程安全测试通过")
}

// TestRegistryUtilityMethods 测试注册器的工具方法
func TestRegistryUtilityMethods(t *testing.T) {
	// 测试命令数量
	count := GlobalRegistry.GetCommandCount()
	if count != 4 {
		t.Errorf("预期命令数量为 4，实际为 %d", count)
	}
	
	// 测试命令列表
	cmdNames := GlobalRegistry.ListCommands()
	if len(cmdNames) != 4 {
		t.Errorf("预期命令列表长度为 4，实际为 %d", len(cmdNames))
	}
	
	// 验证所有预期的命令都在列表中
	expectedCommands := map[string]bool{
		"print.getPrinters":        false,
		"print.getDefaultPrinter":  false,
		"print.setDefaultPrinter":  false,
		"print.testPrint":          false,
	}
	
	for _, cmdName := range cmdNames {
		if _, exists := expectedCommands[cmdName]; exists {
			expectedCommands[cmdName] = true
		}
	}
	
	for cmdName, found := range expectedCommands {
		if !found {
			t.Errorf("命令 '%s' 未在命令列表中找到", cmdName)
		}
	}
	
	t.Log("工具方法测试通过")
}