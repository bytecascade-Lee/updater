// Package config 定义更新器 JSON 配置的结构与校验。
package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Config 顶层配置结构。
type Config struct {
	Version  FlexibleInt `json:"version"`
	Runtime  Runtime     `json:"runtime"`
	Wait     Wait        `json:"wait"`
	Update   Update      `json:"update"`
	Launch   Launch      `json:"launch"`
	Rollback Rollback    `json:"rollback"`
}

// FlexibleInt 兼容 JSON 整数与整数数字字符串（如 "version": "1"）。
type FlexibleInt int

// UnmarshalJSON 实现宽松整数解析。
func (f *FlexibleInt) UnmarshalJSON(data []byte) error {
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexibleInt(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return fmt.Errorf("invalid integer string %q", s)
		}
		*f = FlexibleInt(n)
		return nil
	}
	return fmt.Errorf("invalid integer value: %s", data)
}

// Runtime 更新器进程自身的 UI 与日志行为。
type Runtime struct {
	Headless bool   `json:"headless"`
	LogFile  string `json:"logFile"`
}

// Wait 等待旧进程退出的策略。
type Wait struct {
	PID       int  `json:"pid"`       // 默认 -1；0 或 -1 时跳过等待阶段
	Timeout   *int `json:"timeout"`   // 默认 5000 ms
	ForceKill bool `json:"forceKill"` // 超时后是否强制终止
	Interval  *int `json:"interval"`  // 默认 250 ms
}

// Update 文件替换核心操作。
type Update struct {
	Source          string   `json:"source"`
	Target          string   `json:"target"`
	Preserve        []string `json:"preserve"`
	CleanBeforeCopy *bool    `json:"cleanBeforeCopy"` // 默认 true
	Backup          Backup   `json:"backup"`
}

// Backup 备份配置。
type Backup struct {
	Enabled  bool     `json:"enabled"`
	Location string   `json:"location"` // 为空时自动生成
	Exclude  []string `json:"exclude"`
}

// Launch 新进程启动配置。
type Launch struct {
	Execution Execution `json:"execution"`
	Context   Context   `json:"context"`
	Lifecycle Lifecycle `json:"lifecycle"`
}

// Execution 执行机制。
type Execution struct {
	Mode        string   `json:"mode"` // "direct" | "interpreted"，默认 direct
	Path        string   `json:"path"`
	Interpreter []string `json:"interpreter"`
}

// Context 运行时环境。
type Context struct {
	Workspace string            `json:"workspace"`
	Args      []string          `json:"args"`
	Env       map[string]string `json:"env"`
}

// Lifecycle 更新器驻留行为。
type Lifecycle struct {
	StayAlive     int  `json:"stayAlive"`     // 0 分离退出；-1 无限等待；>0 守护毫秒数
	CaptureOutput bool `json:"captureOutput"` // 仅 stayAlive != 0 时有效
}

// Rollback 失败恢复策略。
type Rollback struct {
	Enabled            bool   `json:"enabled"`
	FallbackExecutable string `json:"fallbackExecutable"`
	MaxAttempts        *int   `json:"maxAttempts"` // 默认 1
}
