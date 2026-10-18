import type { AgentClientId } from "./agent-clients"
import type { AgentInstallTarget } from "./agent-installation-targets"

export type AgentClientEntry = {
  client: AgentClientId
  target: AgentInstallTarget
  title?: "agents.harnessWeb"
}

export const agentClientGroups = [
  {
    title: "agents.codexGroup",
    description: "agents.codexDescription",
    entries: [
      { client: "codex", target: "codex-cli" },
      { client: "codex", target: "codex-desktop" },
    ],
  },
  {
    title: "agents.claudeGroup",
    description: "agents.claudeGroupDescription",
    entries: [
      { client: "claude", target: "claude-code" },
      { client: "claude-desktop", target: "claude-desktop" },
    ],
  },
  {
    title: "agents.opencodeGroup",
    description: "agents.opencodeDescription",
    entries: [
      { client: "opencode", target: "opencode-cli" },
      { client: "opencode", target: "opencode-desktop" },
    ],
  },
  {
    title: "agents.harness",
    description: "agents.harnessDescription",
    entries: [
      { client: "harness-desktop", target: "harness-desktop" },
      { client: "harness-web", target: "harness-cli", title: "agents.harnessWeb" },
    ],
  },
  {
    title: "agents.kilocodeGroup",
    description: "agents.kilocodeDescription",
    entries: [
      { client: "kilocode", target: "kilocode-cli" },
      { client: "kilocode", target: "kilocode-desktop" },
    ],
  },
  {
    title: "agents.qwenCodeGroup",
    description: "agents.qwenCodeDescription",
    entries: [
      { client: "qwen-code", target: "qwen-code-cli" },
      { client: "qwen-code", target: "qwen-code-desktop" },
    ],
  },
  {
    title: "agents.kimiCodeGroup",
    description: "agents.kimiCodeDescription",
    entries: [{ client: "kimi-code", target: "kimi-code-cli" }],
  },
  {
    title: "agents.piGroup",
    description: "agents.piDescription",
    entries: [{ client: "pi", target: "pi-cli" }],
  },
  {
    title: "agents.factoryDroidGroup",
    description: "agents.factoryDroidDescription",
    entries: [{ client: "factory-droid", target: "factory-droid-cli" }],
  },
  {
    title: "agents.gooseGroup",
    description: "agents.gooseDescription",
    entries: [
      { client: "goose", target: "goose-cli" },
      { client: "goose", target: "goose-desktop" },
    ],
  },
  {
    title: "agents.mimoCodeGroup",
    description: "agents.mimoCodeDescription",
    entries: [
      { client: "mimo-code", target: "mimo-code-cli" },
      { client: "mimo-code", target: "mimo-desktop-cn" },
      { client: "mimo-code", target: "mimo-desktop-global" },
    ],
  },
] as const satisfies readonly {
  title: string
  description: string
  entries: readonly AgentClientEntry[]
}[]

export type AgentClientGroup = (typeof agentClientGroups)[number]
