package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"updater/internal/config"
	"updater/internal/logger"
	"updater/internal/ui"
	"updater/pkg/winutil"
)

// ExecuteLaunch 构建并启动新进程，按 lifecycle.stayAlive 决定分离、守护或驻留行为。
func ExecuteLaunch(l *config.Launch, prog *ui.Progress) error {
	logger.Infof("launch stage started: mode=%s path=%s stayAlive=%d captureOutput=%v", l.Execution.Mode, l.Execution.Path, l.Lifecycle.StayAlive, l.Lifecycle.CaptureOutput)
	exe := l.Execution

	// 启动前检查 path 存在
	if _, err := os.Stat(exe.Path); err != nil {
		return fmt.Errorf("launch path %q: %w", exe.Path, err)
	}

	var cmd *exec.Cmd
	if exe.Mode == "interpreted" {
		if len(exe.Interpreter) == 0 {
			return fmt.Errorf("launch.execution.interpreter is empty in interpreted mode")
		}
		// 拼接顺序：[interpreter...] [path] [context.args...]
		args := make([]string, 0, len(exe.Interpreter)+1+len(l.Context.Args))
		args = append(args, exe.Interpreter...)
		args = append(args, exe.Path)
		args = append(args, l.Context.Args...)
		cmd = exec.Command(args[0], args[1:]...)
	} else {
		cmd = exec.Command(exe.Path, l.Context.Args...)
	}

	// workspace 默认为 path 的父目录
	workspace := l.Context.Workspace
	if workspace == "" {
		workspace = filepath.Dir(exe.Path)
	}
	cmd.Dir = workspace
	cmd.Env = append(os.Environ(), envKV(l.Context.Env)...)

	if l.Lifecycle.StayAlive == 0 {
		// 分离模式：不继承控制台，Start 后立即返回，不等待
		cmd.SysProcAttr = winutil.NewDetachedProcAttr()
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start detached process: %w", err)
		}
		logger.Infof("launched detached process (pid %d); updater exits", cmd.Process.Pid)
		prog.Update("[launch] process %d launched (detached)", cmd.Process.Pid)
		return nil
	}

	// 驻留模式：captureOutput 时接管 stdout/stderr（stayAlive=0 时强制忽略）
	if l.Lifecycle.CaptureOutput {
		// capture 模式：子进程输出将写入日志，进度条不再显示
		prog.Finish()
		w := logger.Writer()
		cmd.Stdout = w
		cmd.Stderr = w
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start process: %w", err)
	}
	logger.Infof("launched process (pid %d) in stay-alive mode (%dms)", cmd.Process.Pid, l.Lifecycle.StayAlive)
	prog.Update("[launch] process %d running (stayAlive=%dms)", cmd.Process.Pid, l.Lifecycle.StayAlive)

	if l.Lifecycle.StayAlive < 0 {
		// 无限等待
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("process exited with error: %w", err)
		}
		return nil
	}

	// stayAlive > 0：守护至多指定毫秒数；超时后更新器自行退出，不杀子进程。
	// 期间子进程退出则按退出结果判定（正常退出=成功，报错=失败）。
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("process exited with error before %dms: %w", l.Lifecycle.StayAlive, err)
		}
		return nil
	case <-time.After(time.Duration(l.Lifecycle.StayAlive) * time.Millisecond):
		logger.Infof("stayAlive %dms elapsed; updater exits, process (pid %d) keeps running", l.Lifecycle.StayAlive, cmd.Process.Pid)
		prog.Update("[launch] stayAlive %dms elapsed, updater exits", l.Lifecycle.StayAlive)
		return nil
	}
}

// envKV 将环境变量 map 转为 KEY=VALUE 形式。
func envKV(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}
