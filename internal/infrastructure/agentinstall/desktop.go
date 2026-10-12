package agentinstall

import (
	"encoding/json"
	"slices"
	"strings"
)

func inspectDesktopPackages(output []byte, names []string) Detection {
	var packages []struct{ Name, Version, InstallLocation, Status string }
	if json.Unmarshal(output, &packages) != nil || len(packages) > 8 {
		return unknown("detection_failed")
	}
	if len(packages) == 0 {
		return Detection{State: "not_installed"}
	}
	for _, item := range packages {
		if !slices.Contains(names, item.Name) {
			return unknown("detection_failed")
		}
	}
	// 新旧包可能同时残留；任一受支持的健康包都能确认安装，异常包不遮蔽正常版本。
	for _, item := range packages {
		if validVersion(item.Version) && len(item.InstallLocation) <= 4096 && strings.EqualFold(item.Status, "Ok") {
			return Detection{State: "installed", Version: item.Version, Path: item.InstallLocation}
		}
	}
	return unknown("detection_failed")
}
