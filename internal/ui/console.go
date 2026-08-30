// Package ui 管理更新器进程的 Windows 控制台生命周期。
package ui

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/sys/windows"
)

// attachParentProcess 是 AttachConsole 的"附加到父进程控制台"参数（-1）。
const attachParentProcess = ^uintptr(0)

var (
	consoleOnce sync.Once
	consoleOn   bool
	allocated   bool // 本进程通过 AllocConsole 创建了新窗口（区别于 AttachConsole 附加）
	waitOnExit  bool // 释放控制台前等待按键（仅 CLI 弹窗场景，防一闪而过）

	kernel32          = windows.NewLazySystemDLL("kernel32.dll")
	procAllocConsole  = kernel32.NewProc("AllocConsole")
	procFreeConsole   = kernel32.NewProc("FreeConsole")
	procAttachConsole = kernel32.NewProc("AttachConsole")
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

// attachConsole 附加到指定进程的控制台；pid 为 attachParentProcess 时附加父进程。
func attachConsole(pid uintptr) error {
	r1, _, e1 := procAttachConsole.Call(pid)
	if r1 == 0 {
		return e1
	}
	return nil
}

// stdOutputValid 报告 stdout 是否为有效句柄。
// windowsgui 进程在双击或无重定向启动时标准句柄为 NULL；管道/文件重定向时有效。
func stdOutputValid() bool {
	h, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	return err == nil && h != 0
}

// redirectStdHandles 用 GetStdHandle 重接 os.Stdout/Stderr/Stdin。
// AttachConsole/AllocConsole 后必须重接：Go 的 os.Stdout 固定对应句柄值 1，
// 不会随控制台附加/分配自动生效；当句柄原本有效（如管道）时 GetStdHandle 原样返回，重接为空操作。
func redirectStdHandles() {
	if h, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE); err == nil && h != 0 {
		os.Stdout = os.NewFile(uintptr(h), "/dev/stdout")
	}
	if h, err := windows.GetStdHandle(windows.STD_ERROR_HANDLE); err == nil && h != 0 {
		os.Stderr = os.NewFile(uintptr(h), "/dev/stderr")
	}
	if h, err := windows.GetStdHandle(windows.STD_INPUT_HANDLE); err == nil && h != 0 {
		os.Stdin = os.NewFile(uintptr(h), "/dev/stdin")
	}
}

// ensureConsole 保证存在可用控制台：优先附加父进程控制台复用，失败则弹独立新窗口。
// wait 为 true 时，若最终弹了新窗口，Teardown 释放前会等待按键。
func ensureConsole(wait bool) {
	if err := attachConsole(attachParentProcess); err == nil {
		consoleOn = true
		redirectStdHandles()
		return
	}
	if err := allocConsole(); err != nil {
		// 分配失败：保留现有（无效）标准句柄，输出静默丢弃。
		return
	}
	consoleOn = true
	allocated = true
	waitOnExit = wait
	redirectStdHandles()
}

// SetupCLI 为纯命令行输出（-v/-h/--build-info/无参数/配置错误）准备标准句柄。
// stdout 已是有效句柄（管道/文件重定向，如 Rust 捕获）时不触碰任何窗口；
// 否则附加父终端复用；失败则弹新窗口并在 Teardown 时等待按键。
func SetupCLI() {
	if stdOutputValid() {
		return
	}
	ensureConsole(true)
}

// Setup 根据 headless 决定升级流程的控制台策略。
// headless=true：完全静默，不触碰标准句柄（管道捕获不受影响）。
// headless=false：优先附加父终端复用（终端手动执行场景）；失败则弹独立新窗口。
// 窗口驻留由配置 lifecycle.stayAlive 控制：进程驻留期间窗口不关闭。
func Setup(headless bool) {
	if headless {
		return
	}
	consoleOnce.Do(func() {
		ensureConsole(false)
	})
}

// Teardown 释放已分配的控制台。
// 弹过新窗口且要求驻留时先等待按键；AttachConsole 场景仅解除附加，不影响父控制台。
func Teardown() {
	if !consoleOn {
		return
	}
	if allocated && waitOnExit {
		waitForExitKey()
	}
	freeConsole()
	consoleOn = false
	allocated = false
	waitOnExit = false
}

// waitForExitKey 等待用户按任意键；仅在本进程弹出的控制台窗口上调用。
// 临时关闭行缓冲与回显后读取单字节，实现"任意键"而非"回车"。
func waitForExitKey() {
	h, err := windows.GetStdHandle(windows.STD_INPUT_HANDLE)
	if err != nil || h == 0 {
		return
	}
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return // 非交互输入（如管道），不等待
	}
	windows.SetConsoleMode(h, mode & ^uint32(windows.ENABLE_LINE_INPUT|windows.ENABLE_ECHO_INPUT))
	defer windows.SetConsoleMode(h, mode)
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "按任意键关闭窗口...")
	var b [1]byte
	os.Stdin.Read(b[:])
}
