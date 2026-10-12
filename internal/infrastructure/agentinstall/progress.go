package agentinstall

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Progress 只传递阶段和数值，终端原文、路径及下载 URL 不进入界面。
type Progress struct {
	Phase           string   `json:"phase"`
	Percent         *float64 `json:"percent"`
	DownloadedBytes int64    `json:"downloadedBytes"`
	TotalBytes      int64    `json:"totalBytes"`
}

var (
	byteProgressPattern    = regexp.MustCompile(`(?i)^[\s█▒░▓▏▎▍▌▋▊▉━─]*([0-9]+(?:[.,][0-9]+)?)\s*(B|KB|MB|GB|TB)\s*/\s*([0-9]+(?:[.,][0-9]+)?)\s*(B|KB|MB|GB|TB)\s*$`)
	percentProgressPattern = regexp.MustCompile(`^[\s█▒░▓▏▎▍▌▋▊▉━─]*([0-9]{1,3}(?:\.[0-9]+)?)%\s*$`)
)

// progressWriter 有界读取终端刷新帧，兼容跨 Write 的 ANSI 控制序列与回车更新。
type progressWriter struct {
	report   func(Progress)
	line     []byte
	escape   int
	overflow bool
	osc      []byte
	current  Progress
}

func (w *progressWriter) Write(data []byte) (int, error) {
	for _, value := range data {
		if w.escape != 0 {
			switch w.escape {
			case 1:
				if value == '[' {
					w.escape = 2
				} else if value == ']' {
					w.escape = 3
					w.osc = w.osc[:0]
				} else {
					w.escape = 0
				}
			case 2:
				if value >= 0x40 && value <= 0x7e {
					w.escape = 0
					if value == 'H' || value == 'f' || value == 'G' || value == 'K' {
						w.flush()
					}
				}
			case 3:
				if value == 7 {
					w.escape = 0
					w.flushOSC()
				} else if value == 27 {
					w.escape = 4
				} else if len(w.osc) < 128 {
					w.osc = append(w.osc, value)
				}
			case 4:
				if value == '\\' {
					w.escape = 0
					w.flushOSC()
				} else {
					w.escape = 3
				}
			}
			continue
		}
		if value == 27 {
			w.escape = 1
			continue
		}
		if value == '\r' || value == '\n' {
			w.flush()
			continue
		}
		if value < 32 {
			continue
		}
		if len(w.line) >= 8192 {
			w.overflow = true
			continue
		}
		w.line = append(w.line, value)
	}
	if !w.overflow && w.report != nil {
		if progress, ok := parseProgress(string(w.line)); ok {
			w.emit(progress)
		}
	}
	return len(data), nil
}

func (w *progressWriter) flush() {
	if !w.overflow && w.report != nil {
		if progress, ok := parseProgress(string(w.line)); ok {
			w.emit(progress)
		}
	}
	w.line = w.line[:0]
	w.overflow = false
}

func (w *progressWriter) emit(progress Progress) {
	w.current = progress
	if w.report != nil {
		w.report(progress)
	}
}

func (w *progressWriter) flushOSC() {
	// WinGet 发出的 Windows Terminal 进度指令，不依赖本地化文本或屏幕重绘方式。
	fields := strings.Split(string(w.osc), ";")
	if len(fields) < 3 || fields[0] != "9" || fields[1] != "4" {
		return
	}
	progress := w.current
	if progress.Phase == "" {
		progress.Phase = "preparing"
	}
	if fields[2] == "3" {
		progress.Percent = nil
		w.emit(progress)
		return
	}
	if fields[2] != "1" || len(fields) != 4 {
		return
	}
	value, err := strconv.ParseFloat(fields[3], 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 100 {
		return
	}
	progress.Percent = &value
	w.emit(progress)
}

func parseProgress(line string) (Progress, bool) {
	line = strings.TrimSpace(line)
	if match := byteProgressPattern.FindStringSubmatch(line); len(match) == 5 {
		current, okCurrent := progressBytes(match[1], match[2])
		total, okTotal := progressBytes(match[3], match[4])
		if !okCurrent || !okTotal || total <= 0 || current > total {
			return Progress{}, false
		}
		percent := float64(current) * 100 / float64(total)
		return Progress{Phase: "downloading", Percent: &percent, DownloadedBytes: current, TotalBytes: total}, true
	}
	if match := percentProgressPattern.FindStringSubmatch(line); len(match) == 2 {
		percent, err := strconv.ParseFloat(match[1], 64)
		if err == nil && percent >= 0 && percent <= 100 {
			return Progress{Phase: "installing", Percent: &percent}, true
		}
	}
	lower := strings.ToLower(line)
	if strings.HasPrefix(lower, "downloading ") || strings.HasPrefix(line, "正在下载") || strings.HasPrefix(line, "下载中") {
		return Progress{Phase: "downloading"}, true
	}
	if strings.HasPrefix(lower, "starting package install") || strings.HasPrefix(lower, "installing ") || strings.HasPrefix(line, "正在启动程序包安装") || strings.HasPrefix(line, "正在安装") {
		return Progress{Phase: "installing"}, true
	}
	return Progress{}, false
}

func progressBytes(value, unit string) (int64, bool) {
	number, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", "."), 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 {
		return 0, false
	}
	power := map[string]int{"B": 0, "KB": 1, "MB": 2, "GB": 3, "TB": 4}[strings.ToUpper(unit)]
	bytes := number * math.Pow(1024, float64(power))
	if bytes > 1<<50 {
		return 0, false
	}
	return int64(bytes), true
}
