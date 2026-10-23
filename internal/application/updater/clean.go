package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// helperFile 是每次安装尝试由 appinstall.Launch 复制出的辅助程序副本
// （与自身同规模的 exe）；安装结束后不再被任何流程引用。
var helperFile = regexp.MustCompile(`^runner-[0-9a-f]{32}\.exe$`)

// CleanCache 删除更新目录中确定不再需要的文件：安装过程留下的辅助程序副本
// 和写入中断的临时文件。已下载的安装包、status.json 与安装结果保持不动，
// 以免下次安装强制重新下载或丢失诊断信息。
func CleanCache(directory string) (int64, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("update: read %s: %w", directory, err)
	}
	var freed int64
	for _, entry := range entries {
		if entry.IsDir() || !obsolete(entry.Name()) {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return freed, fmt.Errorf("update: stat %s: %w", entry.Name(), infoErr)
		}
		// 正在运行的辅助进程会占用同名文件；如实报告而不是假装清理成功。
		if removeErr := os.Remove(filepath.Join(directory, entry.Name())); removeErr != nil && !os.IsNotExist(removeErr) {
			return freed, fmt.Errorf("update: remove %s: %w", entry.Name(), removeErr)
		}
		freed += info.Size()
	}
	return freed, nil
}

func obsolete(name string) bool {
	return helperFile.MatchString(name) || strings.HasSuffix(name, ".tmp")
}
