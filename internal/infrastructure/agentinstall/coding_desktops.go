package agentinstall

import "strings"

func codingDesktopMatches(id, display, publisher string) bool {
	var names, publishers []string
	switch id {
	case "harness-desktop":
		names = []string{"DeepSeek Harness"}
		publishers = []string{"DeepSeek"}
	case "kilocode-desktop":
		names = []string{"Kilo Desktop", "Kilo"}
		publishers = []string{"Kilo", "Anaconda"}
	case "qwen-code-desktop":
		names = []string{"Qwen Code", "Qwen Code Desktop"}
		publishers = []string{"Qwen", "Alibaba"}
	case "mimo-desktop-cn":
		names = []string{"Xiaomi MiMo"}
		publishers = []string{"Xiaomi", "MiMo", "小米"}
	case "mimo-desktop-global":
		names = []string{"Xiaomi MiMo AI"}
		publishers = []string{"Xiaomi", "MiMo", "小米"}
	default:
		return false
	}
	nameMatches := false
	for _, name := range names {
		nameMatches = nameMatches || strings.EqualFold(display, name)
	}
	if !nameMatches {
		return false
	}
	for _, prefix := range publishers {
		if strings.HasPrefix(strings.ToLower(publisher), strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}
