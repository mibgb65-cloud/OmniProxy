import type { AgentClientId } from "./agent-clients"

export const agentInstallationTargets = {
  "codex-cli": "agents.install.codexCli",
  "codex-desktop": "agents.install.codexDesktop",
  "claude-code": "agents.install.claudeCode",
  "claude-desktop": "agents.install.claudeDesktop",
  "opencode-cli": "agents.install.opencodeCli",
  "opencode-desktop": "agents.install.opencodeDesktop",
  "harness-cli": "agents.install.harnessCli",
  "harness-desktop": "agents.install.harnessDesktop",
  "kilocode-cli": "agents.install.kilocodeCli",
  "kilocode-desktop": "agents.install.kilocodeDesktop",
  "qwen-code-cli": "agents.install.qwenCodeCli",
  "qwen-code-desktop": "agents.install.qwenCodeDesktop",
  "mimo-code-cli": "agents.install.mimoCodeCli",
  "mimo-desktop-cn": "agents.install.mimoDesktopCn",
  "mimo-desktop-global": "agents.install.mimoDesktopGlobal",
  "kimi-code-cli": "agents.install.kimiCodeCli",
  "pi-cli": "agents.install.piCli",
  "factory-droid-cli": "agents.install.factoryDroidCli",
  "goose-cli": "agents.install.gooseCli",
  "goose-desktop": "agents.install.gooseDesktop",
} as const

export type AgentInstallTarget = keyof typeof agentInstallationTargets
export const clientInstallationTargets: Record<AgentClientId, AgentInstallTarget[]> = {
  codex: ["codex-cli", "codex-desktop"],
  claude: ["claude-code"],
  "claude-desktop": ["claude-desktop"],
  opencode: ["opencode-cli", "opencode-desktop"],
  "harness-web": ["harness-cli"],
  "harness-desktop": ["harness-desktop"],
  kilocode: ["kilocode-cli", "kilocode-desktop"],
  "qwen-code": ["qwen-code-cli", "qwen-code-desktop"],
  "mimo-code": ["mimo-code-cli", "mimo-desktop-cn", "mimo-desktop-global"],
  "kimi-code": ["kimi-code-cli"],
  pi: ["pi-cli"],
  "factory-droid": ["factory-droid-cli"],
  goose: ["goose-cli", "goose-desktop"],
}
