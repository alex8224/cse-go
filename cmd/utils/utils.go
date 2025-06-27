package utils

import "runtime"

// GetOSType 获取当前操作系统类型
func GetOSType() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "darwin":
		return "MacOS"
	case "linux":
		return "Linux"
	default:
		return "Unknown"
	}
}
