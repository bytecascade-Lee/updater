package runner

import (
	"fmt"
	"time"

	"updater/internal/logger"
	"updater/internal/ui"
	"updater/pkg/winutil"
)

// WaitForExit 轮询 pid 进程直到退出；超时后按 forceKill 决定强制终止或失败。
// pid 为 -1 或 0 时直接返回 nil（跳过等待阶段）。
func WaitForExit(pid, timeoutMs, intervalMs int, forceKill bool, prog *ui.Progress) error {
	if pid <= 0 {
		logger.Infof("wait stage skipped: pid=%d", pid)
		return nil
	}
	if timeoutMs < 0 {
		timeoutMs = 0
	}
	if intervalMs <= 0 {
		intervalMs = 250
	}
	logger.Infof("waiting for pid %d to exit (timeout=%dms, interval=%dms, forceKill=%v)", pid, timeoutMs, intervalMs, forceKill)

	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for winutil.ProcessExists(pid) {
		if time.Now().After(deadline) {
			logger.Warnf("wait timeout after %dms: pid %d still alive", timeoutMs, pid)
			if !forceKill {
				return fmt.Errorf("wait timeout: pid %d still alive after %dms", pid, timeoutMs)
			}
			logger.Infof("force killing pid %d", pid)
			prog.Update("[wait] force killing pid %d", pid)
			if err := winutil.KillProcess(pid); err != nil {
				return fmt.Errorf("wait timeout and force kill failed: %w", err)
			}
			logger.Infof("pid %d kill signal sent, waiting for it to fully exit", pid)
			prog.Update("[wait] pid %d terminated, waiting to fully exit", pid)
			// 等待进程彻底退出，确保文件锁释放
			killDeadline := time.Now().Add(5 * time.Second)
			for winutil.ProcessExists(pid) {
				prog.Update("[wait] pid %d still reported alive after kill (%dms remaining)", pid, time.Until(killDeadline).Milliseconds())
				if time.Now().After(killDeadline) {
					return fmt.Errorf("process %d terminated but still reported alive", pid)
				}
				time.Sleep(100 * time.Millisecond)
			}
			logger.Infof("pid %d fully exited", pid)
			prog.Update("[wait] pid %d fully exited", pid)
			return nil
		}
		prog.Update("[wait] pid %d still alive, %dms remaining", pid, time.Until(deadline).Milliseconds())
		logger.Infof("pid %d still alive, keep waiting (remaining %dms)", pid, time.Until(deadline).Milliseconds())
		time.Sleep(time.Duration(intervalMs) * time.Millisecond)
	}
	logger.Infof("pid %d already exited, wait stage done", pid)
	prog.Update("[wait] pid %d already exited", pid)
	return nil
}
