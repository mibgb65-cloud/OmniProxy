package agentconfig

// Clients 是客户端配置、密钥及受管目录共用的精确白名单。
func Clients() []string {
	return []string{"codex", "claude", "claude-desktop", "opencode", "harness-web", "harness-desktop"}
}

func isHarness(client string) bool { return client == "harness-web" || client == "harness-desktop" }

func harnessRef(client string) string {
	if client == "harness-desktop" {
		return "OMNIPROXY_HARNESS_DESKTOP_KEY"
	}
	return "OMNIPROXY_HARNESS_WEB_KEY"
}
