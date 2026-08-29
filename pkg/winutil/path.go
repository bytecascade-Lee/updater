// Package winutil 提供 Windows 平台可复用的底层辅助函数。
package winutil

import (
	"os"
	"path/filepath"
	"strings"
)

// NormPath 归一化路径：清理冗余分隔符并统一小写（Windows 路径大小写不敏感）。
func NormPath(p string) string {
	return strings.ToLower(filepath.Clean(p))
}

// IsPathInside 判断 path 是否位于 dir 内部（含二者相等，与开发手册 4.1 的判定一致）。
func IsPathInside(path, dir string) bool {
	sep := string(os.PathSeparator)
	return strings.HasPrefix(NormPath(path)+sep, NormPath(dir)+sep)
}

// PathInSet 判断 path 是否命中集合中的某条目：精确相等，或位于某条目（目录）之下。
// 集合条目须已通过 NormPath 归一化；nil 集合永远返回 false。
func PathInSet(path string, set map[string]struct{}) bool {
	norm := NormPath(path)
	if _, ok := set[norm]; ok {
		return true
	}
	sep := string(os.PathSeparator)
	for entry := range set {
		if strings.HasPrefix(norm, entry+sep) {
			return true
		}
	}
	return false
}
