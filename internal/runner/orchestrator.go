// Package runner 编排更新流程：wait → update → launch →（可选 rollback）。
package runner

import (
	"updater/internal/config"
	"updater/internal/logger"
)

// Run 按序执行更新流程并返回进程退出码。
// 0=成功；2=wait 失败；3=update 失败；4=launch 失败（含回滚成功的情况）；5=rollback 失败。
func Run(cfg *config.Config) int {
	// 1. wait
	if err := WaitForExit(cfg.Wait.PID, *cfg.Wait.Timeout, *cfg.Wait.Interval, cfg.Wait.ForceKill); err != nil {
		logger.Errorf("wait stage failed: %v", err)
		return 2
	}

	// 2. update
	backupLocation, err := ExecuteUpdate(&cfg.Update)
	if err != nil {
		logger.Errorf("update stage failed: %v", err)
		return 3
	}

	// 3. launch
	if err := ExecuteLaunch(&cfg.Launch); err != nil {
		logger.Errorf("launch stage failed: %v", err)
		// 4. rollback（launch 失败，且启用了备份与回滚）
		if cfg.Rollback.Enabled && cfg.Update.Backup.Enabled {
			logger.Warn("launch failed; attempting rollback")
			if rbErr := ExecuteRollback(cfg, backupLocation); rbErr != nil {
				logger.Errorf("rollback stage failed: %v", rbErr)
				return 5
			}
			logger.Warn("rollback completed; new process was not started")
			return 4
		}
		return 4
	}
	return 0
}
