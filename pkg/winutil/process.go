package winutil

import (
	"errors"
	"fmt"
	"syscall"

	"golang.org/x/sys/windows"
)

// STILL_ACTIVE 是进程仍在运行时的退出码（GetExitCodeProcess 返回 0x103）。
// golang.org/x/sys 未导出该常量，此处自行定义。
const STILL_ACTIVE = 259

// ProcessExists 检查 pid 对应的进程是否存在。
func ProcessExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		// ERROR_INVALID_PARAMETER 表示进程不存在；其余错误（如拒绝访问）视为存在。
		return !errors.Is(err, windows.ERROR_INVALID_PARAMETER)
	}
	defer windows.CloseHandle(h)
	// OpenProcess 在进程被终止后仍可能成功（对象残留），须用退出码判断真正存活。
	var exitCode uint32
	if err := windows.GetExitCodeProcess(h, &exitCode); err != nil {
		return false
	}
	return exitCode == STILL_ACTIVE
}

// KillProcess 强制终止 pid 对应的进程。
func KillProcess(pid int) error {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return fmt.Errorf("open process %d: %w", pid, err)
	}
	defer windows.CloseHandle(h)
	if err := windows.TerminateProcess(h, 1); err != nil {
		return fmt.Errorf("terminate process %d: %w", pid, err)
	}
	return nil
}

// NewDetachedProcAttr 返回分离启动属性：不继承调用方控制台，也不创建新窗口。
func NewDetachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NO_WINDOW,
	}
}
