package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"updater/internal/config"
	"updater/internal/logger"
	"updater/internal/ui"
	"updater/pkg/winutil"
)

// ExecuteRollback 还原备份并启动旧进程（始终分离，忽略生命周期配置，重试 maxAttempts 次）。
func ExecuteRollback(cfg *config.Config, backupLocation string, prog *ui.Progress) error {
	logger.Infof("rollback stage started: restore=%s fallback=%q maxAttempts=%d", backupLocation, cfg.Rollback.FallbackExecutable, *cfg.Rollback.MaxAttempts)
	if backupLocation == "" {
		return fmt.Errorf("backup location is empty; nothing to restore")
	}
	if _, err := os.Stat(backupLocation); err != nil {
		return fmt.Errorf("backup location %q: %w", backupLocation, err)
	}

	// 1. 还原：备份为完整复制，回滚亦为完整复制
	prog.Update("[rollback] restoring %s -> %s", backupLocation, cfg.Update.Target)
	if err := copyDir(backupLocation, cfg.Update.Target, nil); err != nil {
		return fmt.Errorf("restore %s -> %s: %w", backupLocation, cfg.Update.Target, err)
	}
	logger.Infof("restored %s -> %s", backupLocation, cfg.Update.Target)
	prog.Update("[rollback] backup restored")

	// 2. 启动旧进程（fallbackExecutable 为空时仅还原，不启动任何进程）
	fallback := cfg.Rollback.FallbackExecutable
	if fallback == "" {
		logger.Warn("fallbackExecutable is empty; backup restored, no process launched")
		return nil
	}
	if _, err := os.Stat(fallback); err != nil {
		return fmt.Errorf("fallback executable %q: %w", fallback, err)
	}
	attempts := 1
	if cfg.Rollback.MaxAttempts != nil && *cfg.Rollback.MaxAttempts > 0 {
		attempts = *cfg.Rollback.MaxAttempts
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		prog.Update("[rollback] launching fallback (attempt %d/%d)", i+1, attempts)
		cmd := exec.Command(fallback)
		cmd.Dir = filepath.Dir(fallback)
		cmd.SysProcAttr = winutil.NewDetachedProcAttr()
		if err := cmd.Start(); err != nil {
			lastErr = err
			logger.Errorf("fallback launch attempt %d/%d failed: %v", i+1, attempts, err)
			continue
		}
		logger.Infof("fallback process launched (pid %d)", cmd.Process.Pid)
		return nil
	}
	return fmt.Errorf("fallback executable %q failed after %d attempts: %w", fallback, attempts, lastErr)
}
