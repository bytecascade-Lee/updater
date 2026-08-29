// Package ui 管理更新器进程的 Windows 控制台生命周期。
package ui

import (
	"os"
	"sync"

	"golang.org/x/sys/windows"
)

var (
	consoleOnce sync.Once
	consoleOn   bool

	kernel32         = windows.NewLazySystemDLL("kernel32.dll")
	procAllocConsole = kernel32.NewProc("AllocConsole")
	procFreeConsole  = kernel32.NewProc("FreeConsole")
)

// allocConsole 分配新控制台。golang.org/x/sys 新版本未导出 AllocConsole，此处动态调用。
func allocConsole() error {
	r1, _, e1 := procAllocConsole.Call()
	if r1 == 0 {
		return e1
	}
	return nil
}

// freeConsole 释放当前控制台。
func freeConsole() error {
	r1, _, e1 := procFreeConsole.Call()
	if r1 == 0 {
		return e1
	}
	return nil
}

// Setup 根据 headless 决定是否分配新控制台并重定向标准句柄。
func Setup(headless bool) {
	if headless {
		return
	}
	consoleOnce.Do(func() {
		if err := allocConsole(); err != nil {
			// 已有控制台或分配失败：保留现有标准句柄。
			return
		}
		consoleOn = true
		if h, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE); err == nil && h != 0 {
			os.Stdout = os.NewFile(uintptr(h), "/dev/stdout")
		}
		if h, err := windows.GetStdHandle(windows.STD_ERROR_HANDLE); err == nil && h != 0 {
			os.Stderr = os.NewFile(uintptr(h), "/dev/stderr")
		}
		if h, err := windows.GetStdHandle(windows.STD_INPUT_HANDLE); err == nil && h != 0 {
			os.Stdin = os.NewFile(uintptr(h), "/dev/stdin")
		}
	})
}

// Teardown 释放已分配的控制台。
func Teardown() {
	if consoleOn {
		freeConsole()
		consoleOn = false
	}
}
