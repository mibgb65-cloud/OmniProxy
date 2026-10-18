package agentinstall

import "strings"

func gatewayAgentVersion(command, text string) string {
	prefixes := map[string][]string{
		"kimi":  {"Kimi Code CLI ", "Kimi Code ", "kimi-cli, version ", "kimi, version ", "kimi "},
		"pi":    {"pi version ", "pi ", "Pi "},
		"droid": {"Droid CLI ", "droid version ", "droid ", "Droid "},
		"goose": {"goose version ", "goose "},
	}
	if validVersion(text) {
		return text
	}
	for _, prefix := range prefixes[command] {
		if strings.HasPrefix(text, prefix) {
			version := strings.TrimPrefix(strings.TrimPrefix(text, prefix), "v")
			if validVersion(version) {
				return version
			}
		}
	}
	return ""
}
