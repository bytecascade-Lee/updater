// 更新器 CLI：读取 JSON 配置文件，执行 等待 → 更新 → 启动 →（可选回滚）流程。
package main

import (
	"fmt"
	"io"
	"os"

	"updater/internal/config"
	"updater/internal/logger"
	"updater/internal/runner"
	"updater/internal/ui"
)

const updaterVersion = "1.0.0"

func main() {
	exitCode := 0
	defer func() {
		if r := recover(); r != nil {
			exitCode = 99
		}
		os.Exit(exitCode)
	}()

	args := os.Args[1:]
	if len(args) == 0 {
		usage(os.Stderr)
		exitCode = 1
		return
	}
	switch args[0] {
	case "--help", "-h":
		usage(os.Stdout)
		return
	case "--version", "-v":
		fmt.Fprintf(os.Stdout, "updater %s\n", updaterVersion)
		return
	}

	cfg, err := config.Load(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		exitCode = 1
		return
	}

	ui.Setup(cfg.Runtime.Headless)
	closer, err := logger.Init(cfg.Runtime.LogFile, cfg.Runtime.Headless)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger init error: %v\n", err)
		exitCode = 1
		return
	}
	if closer != nil {
		defer closer.Close()
	}

	exitCode = runner.Run(cfg)
	ui.Teardown()
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage: updater <config.json>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  -h, --help     Show this help message")
	fmt.Fprintln(w, "  -v, --version  Show version")
}
