import type { AgentClient } from "../../services/agents"

export const agentClientIds = [
  "codex",
  "claude",
  "claude-desktop",
  "opencode",
  "harness-web",
  "harness-desktop",
] as const
export type AgentClientId = (typeof agentClientIds)[number]

export const agentClients = {
  codex: { title: "agents.codex", description: "agents.codexDescription", icon: "codex.svg" },
  claude: {
    title: "agents.claude",
    description: "agents.claudeDescription",
    icon: "claudecode-color.svg",
  },
  "claude-desktop": {
    title: "agents.claude-desktop",
    description: "agents.desktopDescription",
    icon: "claude-color.svg",
  },
  opencode: {
    title: "agents.opencode",
    description: "agents.opencodeDescription",
    icon: "opencode.svg",
  },
  "harness-web": {
    title: "agents.harnessWeb",
    description: "agents.harnessWebDescription",
    icon: "",
  },
  "harness-desktop": {
    title: "agents.harnessDesktop",
    description: "agents.harnessDesktopDescription",
    icon: "",
  },
} as const

export function isAgentClientId(value: unknown): value is AgentClientId {
  return typeof value === "string" && agentClientIds.some((id) => id === value)
}

export function agentStatusKey(item?: AgentClient) {
  if (!item) return "agents.unavailable"
  if (item.errorCode) return "agents.needsAttention"
  if (item.configured) return "agents.configured"
  return item.canRestore ? "agents.disabled" : "agents.idle"
}
