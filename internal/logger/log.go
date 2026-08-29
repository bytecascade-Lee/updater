// Package logger 封装 logrus，支持文件输出与控制台双写（由 headless 控制）。
package logger

import (
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

// Init 初始化日志输出。
// headless=true 只写 logFile；false 双写控制台与 logFile；
// logFile 为空且 headless=true 时禁用全部输出。
// 返回日志文件（调用方负责关闭），未打开文件时返回 nil。
func Init(logFile string, headless bool, clearLine func()) (io.Closer, error) {
	log = logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		DisableColors: true,
	})
	if headless && logFile == "" {
		log.SetOutput(io.Discard)
		return nil, nil
	}
	writers := make([]io.Writer, 0, 2)
	var file *os.File
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("open log file %q: %w", logFile, err)
		}
		file = f
		writers = append(writers, file)
	}
	if !headless {
		writers = append(writers, os.Stdout)
	}
	if len(writers) == 0 {
		writers = append(writers, io.Discard)
	}
	log.SetOutput(io.MultiWriter(writers...))
	if clearLine != nil {
		log.AddHook(&lineHook{clear: clearLine})
	}
	return file, nil
}

// Writer 返回一个将写入内容作为日志记录的 writer（用于捕获子进程 stdout/stderr）。
func Writer() io.Writer {
	if log == nil {
		return io.Discard
	}
	return log.Writer()
}

// Info 记录 Info 级别日志。
func Info(args ...interface{}) { log.Info(args...) }

// Infof 记录 Info 级别日志（格式化）。
func Infof(format string, args ...interface{}) { log.Infof(format, args...) }

// Warn 记录 Warn 级别日志。
func Warn(args ...interface{}) { log.Warn(args...) }

// Warnf 记录 Warn 级别日志（格式化）。
func Warnf(format string, args ...interface{}) { log.Warnf(format, args...) }

// Error 记录 Error 级别日志。
func Error(args ...interface{}) { log.Error(args...) }

// Errorf 记录 Error 级别日志（格式化）。
func Errorf(format string, args ...interface{}) { log.Errorf(format, args...) }

// lineHook 在每条日志输出前调用清行回调，使进度条与日志不交错。
type lineHook struct {
	clear func()
}

// Levels 返回所有日志级别。
func (h *lineHook) Levels() []logrus.Level { return logrus.AllLevels }

// Fire 在日志写出前清除进度条所在行。
func (h *lineHook) Fire(*logrus.Entry) error {
	if h.clear != nil {
		h.clear()
	}
	return nil
}
