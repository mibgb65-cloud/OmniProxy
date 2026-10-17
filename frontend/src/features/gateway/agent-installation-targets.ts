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
}
