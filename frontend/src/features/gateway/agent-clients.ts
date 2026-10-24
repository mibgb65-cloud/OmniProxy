import type { TranslationKey } from "../../i18n/translation"
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

// Codex 的 CLI、桌面应用与 IDE 扩展共用同一入口前缀，网关只能按客户端自己声明的
// 来源标识区分它们；请求记录里用这几项替代笼统的「Codex CLI / Desktop」。
export const requestAgentTitleKeys = {
  "codex-cli": "agents.codex-cli",
  "codex-desktop": "agents.codex-desktop",
  "codex-vscode": "agents.codex-vscode",
} as const

/** 请求归属的展示键：运行形态细化优先，其余沿用客户端目录。 */
export function requestAgentTitleKey(client?: string): TranslationKey | null {
  if (client && client in requestAgentTitleKeys) {
    return requestAgentTitleKeys[client as keyof typeof requestAgentTitleKeys]
  }
  return agentTitleKey(client)
}

export function agentStatusKey(item?: AgentClient) {
  if (!item) return "agents.unavailable"
  if (item.errorCode) return "agents.needsAttention"
  if (item.configured) return "agents.configured"
  return item.canRestore ? "agents.disabled" : "agents.idle"
}
