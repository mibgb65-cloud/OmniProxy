package agentconfig

// Clients 是客户端配置、密钥及受管目录共用的精确白名单。
func Clients() []string {
	return []string{"codex", "claude", "claude-desktop", "opencode", "harness-web", "harness-desktop", "kilocode", "qwen-code", "mimo-code", "kimi-code", "pi", "factory-droid", "goose"}
}

func isCompatibleClient(client string) bool {
	return client == "opencode" || client == "kilocode" || client == "mimo-code"
}

func isChatClient(client string) bool {
	return isCompatibleClient(client) || isHarness(client) || client == "qwen-code" || isGatewayAgent(client)
}

func isGatewayAgent(client string) bool {
	return client == "kimi-code" || client == "pi" || client == "factory-droid" || client == "goose"
}

func isBundledClient(client string) bool { return client == "pi" || client == "goose" }

func isHarness(client string) bool { return client == "harness-web" || client == "harness-desktop" }

func harnessRef(client string) string {
	if client == "harness-desktop" {
		return "OMNIPROXY_HARNESS_DESKTOP_KEY"
	}
	return "OMNIPROXY_HARNESS_WEB_KEY"
}
