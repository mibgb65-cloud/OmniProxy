//go:build windows

package appinstall

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

const installerDirectoryResultName = "omniproxy-data-dir.txt"

func writeInstallerDirectoryResult(executable, directory string) error {
	if !filepath.IsAbs(executable) || directory == "" || len(directory) > 4096 || strings.ContainsAny(directory, "\r\n\x00") {
		return errors.New("installer: invalid_directory_result")
	}
	units := utf16.Encode([]rune(directory + "\r\n"))
	// 默认 NSIS 字符串容量为 1024（含终止符）。超长目录必须显式失败，不能被截断后写回。
	if len(units) >= 1024 {
		return errors.New("installer: directory_result_too_long")
	}
	// nsExec 的文本解码受系统代码页影响。安装器专用结果固定在助手旁，
	// 使用无 BOM 的 UTF-16LE；不接受调用方指定任意输出路径，也不覆盖已有文件。
	path := filepath.Join(filepath.Dir(executable), installerDirectoryResultName)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	data := make([]byte, len(units)*2)
	for index, unit := range units {
		binary.LittleEndian.PutUint16(data[index*2:], unit)
	}
	_, writeErr := file.Write(data)
	return errors.Join(writeErr, file.Close())
}
