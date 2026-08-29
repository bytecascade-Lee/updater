package runner

import (
	"fmt"
	"time"

	"updater/pkg/winutil"
)

// WaitForExit 轮询 pid 进程直到退出；超时后按 forceKill 决定强制终止或失败。
// pid 为 -1 或 0 时直接返回 nil（跳过等待阶段）。
func WaitForExit(pid, timeoutMs, intervalMs int, forceKill bool) error {
	if pid <= 0 {
		return nil
	}
	if timeoutMs < 0 {
		timeoutMs = 0
	}
	if intervalMs <= 0 {
		intervalMs = 250
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for winutil.ProcessExists(pid) {
		if time.Now().After(deadline) {
			if !forceKill {
				return fmt.Errorf("wait timeout: pid %d still alive after %dms", pid, timeoutMs)
			}
			if err := winutil.KillProcess(pid); err != nil {
				return fmt.Errorf("wait timeout and force kill failed: %w", err)
			}
			// 等待进程彻底退出，确保文件锁释放
			killDeadline := time.Now().Add(5 * time.Second)
			for winutil.ProcessExists(pid) {
				if time.Now().After(killDeadline) {
					return fmt.Errorf("process %d terminated but still reported alive", pid)
				}
				time.Sleep(100 * time.Millisecond)
			}
			return nil
		}
		time.Sleep(time.Duration(intervalMs) * time.Millisecond)
	}
	return nil
}
