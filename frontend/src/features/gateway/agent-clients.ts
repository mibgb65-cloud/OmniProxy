import type { AgentClient } from "../../services/agents"

export const agentClientIds = [
  "codex",
  "claude",
  "claude-desktop",
  "opencode",
  "harness-web",
  "harness-desktop",
  "kilocode",
  "qwen-code",
  "mimo-code",
  "kimi-code",
  "pi",
  "factory-droid",
  "goose",
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
  kilocode: {
    title: "agents.kilocode",
    description: "agents.kilocodeDescription",
    icon: "kilocode.svg",
  },
  "qwen-code": {
    title: "agents.qwenCode",
    description: "agents.qwenCodeDescription",
    icon: "qwen-color.svg",
  },
  "mimo-code": { title: "agents.mimoCode", description: "agents.mimoCodeDescription", icon: "" },
  "kimi-code": { title: "agents.kimiCode", description: "agents.kimiCodeDescription", icon: "" },
  pi: { title: "agents.pi", description: "agents.piDescription", icon: "pi.svg" },
  "factory-droid": {
    title: "agents.factoryDroid",
    description: "agents.factoryDroidDescription",
    icon: "factory.svg",
  },
  goose: { title: "agents.goose", description: "agents.gooseDescription", icon: "goose.svg" },
} as const

export function isAgentClientId(value: unknown): value is AgentClientId {
  return typeof value === "string" && agentClientIds.some((id) => id === value)
}

// 请求归属展示用标题键；无法归属的请求（直接调用、历史记录、自定义客户端）返回 null。
export type AgentTitleKey = (typeof agentClients)[AgentClientId]["title"]
export function agentTitleKey(client?: string): AgentTitleKey | null {
  return isAgentClientId(client) ? agentClients[client].title : null
}

export function agentStatusKey(item?: AgentClient) {
  if (!item) return "agents.unavailable"
  if (item.errorCode) return "agents.needsAttention"
  if (item.configured) return "agents.configured"
  return item.canRestore ? "agents.disabled" : "agents.idle"
}
