package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"updater/pkg/winutil"
)

// Load 读取配置文件，应用默认值并执行校验。
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// applyDefaults 填充文档规定的默认值。
func (c *Config) applyDefaults() {
	if c.Wait.Timeout == nil {
		v := 5000
		c.Wait.Timeout = &v
	}
	if c.Wait.Interval == nil {
		v := 250
		c.Wait.Interval = &v
	}
	if c.Update.CleanBeforeCopy == nil {
		v := true
		c.Update.CleanBeforeCopy = &v
	}
	if c.Launch.Execution.Mode == "" {
		c.Launch.Execution.Mode = "direct"
	}
	if c.Rollback.MaxAttempts == nil {
		v := 1
		c.Rollback.MaxAttempts = &v
	}
}

// Validate 校验版本号、必填字段、绝对路径与 backup.location 位置。
func (c *Config) Validate() error {
	switch v := int(c.Version); {
	case v == 1:
		// 当前支持版本
	case v > 1:
		return fmt.Errorf("unsupported config version %d: please upgrade the updater tool", v)
	default:
		return fmt.Errorf("invalid config version %d: expected 1", v)
	}

	if c.Update.Source == "" {
		return errors.New("update.source is required")
	}
	if c.Update.Target == "" {
		return errors.New("update.target is required")
	}
	if len(c.Update.Preserve) == 0 {
		return errors.New("update.preserve must be a non-empty array of absolute paths")
	}
	if c.Launch.Execution.Path == "" {
		return errors.New("launch.execution.path is required")
	}
	if m := c.Launch.Execution.Mode; m != "direct" && m != "interpreted" {
		return fmt.Errorf("launch.execution.mode must be \"direct\" or \"interpreted\", got %q", m)
	}

	// 绝对路径强制
	for _, f := range []struct {
		name string
		path string
	}{
		{"update.source", c.Update.Source},
		{"update.target", c.Update.Target},
		{"launch.execution.path", c.Launch.Execution.Path},
	} {
		if !filepath.IsAbs(f.path) {
			return fmt.Errorf("%s must be an absolute path: %q", f.name, f.path)
		}
	}
	for _, p := range c.Update.Preserve {
		if !filepath.IsAbs(p) {
			return fmt.Errorf("update.preserve entries must be absolute paths: %q", p)
		}
	}
	if c.Update.Backup.Location != "" {
		if !filepath.IsAbs(c.Update.Backup.Location) {
			return fmt.Errorf("update.backup.location must be an absolute path: %q", c.Update.Backup.Location)
		}
		if winutil.IsPathInside(c.Update.Backup.Location, c.Update.Target) {
			return errors.New("update.backup.location cannot be inside target")
		}
	}
	if c.Rollback.FallbackExecutable != "" && !filepath.IsAbs(c.Rollback.FallbackExecutable) {
		return fmt.Errorf("rollback.fallbackExecutable must be an absolute path: %q", c.Rollback.FallbackExecutable)
	}
	if c.Runtime.LogFile != "" && !filepath.IsAbs(c.Runtime.LogFile) {
		return fmt.Errorf("runtime.logFile must be an absolute path: %q", c.Runtime.LogFile)
	}
	// workspace 为空时由 launch 阶段填充默认值（path 的父目录）；非空则必须为绝对路径
	if c.Launch.Context.Workspace != "" && !filepath.IsAbs(c.Launch.Context.Workspace) {
		return fmt.Errorf("launch.context.workspace must be an absolute path: %q", c.Launch.Context.Workspace)
	}
	return nil
}
