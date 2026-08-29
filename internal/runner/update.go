package runner

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"updater/internal/config"
	"updater/internal/logger"
	"updater/internal/ui"
	"updater/pkg/winutil"
)

// ExecuteUpdate 执行 update 阶段：自检警报 → 备份 → 清理 → 拷贝（顺序不可颠倒）。
// 返回实际使用的备份目录（备份未启用时为空字符串）。
func ExecuteUpdate(upd *config.Update, prog *ui.Progress) (string, error) {
	logger.Infof("update stage started: source=%s target=%s cleanBeforeCopy=%v backupEnabled=%v", upd.Source, upd.Target, *upd.CleanBeforeCopy, upd.Backup.Enabled)
	self, _ := os.Executable()

	// 1. 自检：updater 自身位于 target 内时仅记录警告，不阻塞流程
	if winutil.IsPathInside(self, upd.Target) {
		logger.Warnf("updater executable %q is inside target %q; it will be skipped in backup and clean", self, upd.Target)
	}

	// 2. 备份
	backupLocation := ""
	if upd.Backup.Enabled {
		backupLocation = upd.Backup.Location
		if backupLocation == "" {
			backupLocation = filepath.Join(filepath.Dir(upd.Target), filepath.Base(upd.Target)+".bak-"+time.Now().Format("20060102150405"))
			logger.Infof("backup location not set, using %q", backupLocation)
		}
		prog.Update("[update] backing up %s -> %s", upd.Target, backupLocation)
		if err := copyDir(upd.Target, backupLocation, buildExcludeSet(upd.Backup.Exclude, self)); err != nil {
			return "", fmt.Errorf("backup %s -> %s: %w", upd.Target, backupLocation, err)
		}
		logger.Infof("backup completed: %s -> %s", upd.Target, backupLocation)
		prog.Update("[update] backup done: %s", filepath.Base(backupLocation))
	}

	// 3. 清理
	if upd.CleanBeforeCopy == nil || *upd.CleanBeforeCopy {
		prog.Update("[update] cleaning %s (preserving %d entries)", upd.Target, len(upd.Preserve))
		if err := cleanTarget(upd.Target, upd.Preserve, self); err != nil {
			return backupLocation, fmt.Errorf("clean %s: %w", upd.Target, err)
		}
		logger.Infof("clean completed: %s", upd.Target)
	}

	// 4. 拷贝
	prog.Update("[update] copying %s -> %s", upd.Source, upd.Target)
	if err := syncSourceToTarget(upd.Source, upd.Target, upd.Preserve); err != nil {
		return backupLocation, fmt.Errorf("copy %s -> %s: %w", upd.Source, upd.Target, err)
	}
	logger.Infof("copy completed: %s -> %s", upd.Source, upd.Target)
	return backupLocation, nil
}

// buildExcludeSet 构建备份排除集合：用户 exclude 列表 + updater 自身（隐式注入）。
func buildExcludeSet(exclude []string, self string) map[string]struct{} {
	set := make(map[string]struct{}, len(exclude)+1)
	for _, e := range exclude {
		set[winutil.NormPath(e)] = struct{}{}
	}
	if self != "" {
		set[winutil.NormPath(self)] = struct{}{}
	}
	return set
}

// copyDir 递归复制 src 到 dst；命中 excludeSet 的路径被跳过（目录整树跳过）。
func copyDir(src, dst string, excludeSet map[string]struct{}) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if winutil.PathInSet(path, excludeSet) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

// copyFile 复制单个文件。
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// cleanTarget 删除 target 下的全部内容，保留命中 preserve 的路径与 updater 自身。
func cleanTarget(target string, preserve []string, self string) error {
	preserveSet := make(map[string]struct{}, len(preserve))
	for _, p := range preserve {
		preserveSet[winutil.NormPath(p)] = struct{}{}
	}
	return removeTree(target, preserveSet, winutil.NormPath(self))
}

// removeTree 递归删除 dir 下的内容；命中保留集合的路径不删除（目录保留整树）。
// dir 自身不删除；仅当其被清空后由父级删除。
func removeTree(dir string, preserveSet map[string]struct{}, self string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		if winutil.NormPath(p) == self || winutil.PathInSet(p, preserveSet) {
			continue
		}
		if e.IsDir() {
			if err := removeTree(p, preserveSet, self); err != nil {
				return err
			}
			// 递归后目录为空则删除；非空（含保留子项）则保留
			if err := os.Remove(p); err != nil && !errors.Is(err, syscall.ERROR_DIR_NOT_EMPTY) {
				return err
			}
		} else {
			if err := os.Remove(p); err != nil {
				return err
			}
		}
	}
	return nil
}

// syncSourceToTarget 将 source 同步到 target；目标路径命中 preserve 且已存在时跳过写入。
func syncSourceToTarget(source, target string, preserve []string) error {
	preserveSet := make(map[string]struct{}, len(preserve))
	for _, p := range preserve {
		preserveSet[winutil.NormPath(p)] = struct{}{}
	}
	return filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(target, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		if winutil.PathInSet(dst, preserveSet) {
			if _, err := os.Stat(dst); err == nil {
				logger.Warnf("skip writing %s: preserved and already exists", dst)
				return nil
			}
		}
		return copyFile(path, dst)
	})
}
