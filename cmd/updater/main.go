// 更新器 CLI：读取 JSON 配置文件，执行 等待 → 更新 → 启动 →（可选回滚）流程。
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"updater/internal/config"
	"updater/internal/logger"
	"updater/internal/runner"
	"updater/internal/ui"
)

// 版本与构建信息由构建脚本通过 ldflags 注入：
var (
	Version        = "0.1.0-dev"
	BuildTime      = ""
	ShortHash      = ""
	GitCommitCount = ""
	GitCommitTime  = ""
	Branch         = ""
)

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
		ui.SetupCLI()
		usage(os.Stderr)
		ui.Teardown()
		exitCode = 1
		return
	}
	switch args[0] {
	case "--help", "-h":
		ui.SetupCLI()
		usage(os.Stdout)
		ui.Teardown()
		return
	case "--version", "-v":
		ui.SetupCLI()
		fmt.Fprintln(os.Stdout, cleanVersion())
		ui.Teardown()
		return
	case "--build-info":
		ui.SetupCLI()
		fmt.Fprintln(os.Stdout, buildInfoJSON())
		ui.Teardown()
		return
	}

	cfg, err := config.Load(args[0])
	if err != nil {
		ui.SetupCLI()
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		ui.Teardown()
		exitCode = 1
		return
	}

	ui.Setup(cfg.Runtime.Headless)
	prog := ui.NewProgress(!cfg.Runtime.Headless)
	closer, err := logger.Init(cfg.Runtime.LogFile, cfg.Runtime.Headless, prog.Clear)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger init error: %v\n", err)
		exitCode = 1
		return
	}
	if closer != nil {
		defer closer.Close()
	}
	logger.Infof("config loaded: version=%d target=%s", int(cfg.Version), cfg.Update.Target)

	exitCode = runner.Run(cfg, prog)
	ui.Teardown()
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage: updater <config.json>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  -h, --help     Show this help message")
	fmt.Fprintln(w, "  -v, --version    Show version (e.g. 0.1.0)")
	fmt.Fprintln(w, "  --build-info     Show build info as JSON")
}

// cleanVersion 返回不含 v 前缀与 + 构建元数据的版本号（如 0.1.0 / 0.2.3-rc.1），
// 供 -v 输出与主程序捕获。
func cleanVersion() string {
	v := strings.TrimPrefix(Version, "v")
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	return v
}

// buildInfoJSON 输出构建信息（JSON 格式）。
func buildInfoJSON() string {
	count := 0
	if n, err := strconv.Atoi(GitCommitCount); err == nil {
		count = n
	}
	info := map[string]interface{}{
		"version":        cleanVersion(),
		"fullVersion":    strings.TrimPrefix(Version, "v"),
		"buildTime":      BuildTime,
		"shortHash":      ShortHash,
		"gitCommitCount": count,
		"gitCommitTime":  GitCommitTime,
		"branch":         Branch,
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"version":%q}`, cleanVersion())
	}
	return string(data)
}
