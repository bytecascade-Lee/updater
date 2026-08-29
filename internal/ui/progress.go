package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Progress 是固定在输出最下方的状态行，通过 \r 原地覆盖更新。
// headless 或无控制台场景（enabled=false）下所有操作均为空操作。
type Progress struct {
	enabled bool
	done    bool
	text    string
}

// NewProgress 创建进度条；enabled 控制是否实际输出（headless=true 时传 false）。
func NewProgress(enabled bool) *Progress {
	return &Progress{enabled: enabled}
}

// Update 更新进度条文本（不换行，覆盖当前行）。
func (p *Progress) Update(format string, args ...interface{}) {
	if p == nil || !p.enabled || p.done {
		return
	}
	p.text = fmt.Sprintf(format, args...)
	fmt.Print("\r" + p.text)
}

// Clear 清除进度条所在行（日志输出前调用，避免日志与进度条交错）。
func (p *Progress) Clear() {
	if p == nil || !p.enabled || p.done || len(p.text) == 0 {
		return
	}
	// 按字符数填充空白以覆盖整行；末尾多留余量防止残留
	fmt.Print("\r" + strings.Repeat(" ", utf8.RuneCountInString(p.text)+16) + "\r")
	p.text = ""
}

// Finish 终止进度条：清除所在行并换行；之后 Update/Clear 均不再输出。
func (p *Progress) Finish() {
	if p == nil || !p.enabled {
		return
	}
	p.Clear()
	p.done = true
	fmt.Println()
}
