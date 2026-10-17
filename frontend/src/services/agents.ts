import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type { Input, Item } from "../../bindings/omniproxy/internal/application/agentconfig"

export type AgentClient = Item
export type AgentInput = Input
export const listAgentClients = () => AppService.AgentClients()
export const getConfiguredAgentModels = (client: string) => AppService.AgentConfiguredModels(client)
export type ConfiguredAgentCatalog = Awaited<ReturnType<typeof getConfiguredAgentModels>>
export async function rereadAgentConfig(client: string): Promise<AgentClient> {
  if (
    ![
      "codex",
      "claude",
      "claude-desktop",
      "opencode",
      "harness-web",
      "harness-desktop",
      "kilocode",
      "qwen-code",
      "mimo-code",
    ].includes(client)
  )
    throw new Error("agent: invalid_client")
  const items = await listAgentClients()
  const item = items?.find((item) => item.client === client)
  if (!item) throw new Error("agent: unavailable")
  return item
}
export const previewAgentConfig = (input: Input) => AppService.AgentPreview(input)
export const previewAgentPlan = (input: Input) => AppService.AgentPreviewPlan(input)
export type AgentPlan = Awaited<ReturnType<typeof previewAgentPlan>>
export const applyAgentConfig = (input: Input) => AppService.AgentApply(input)
export const restoreAgentConfig = (client: string) => AppService.AgentRestore(client)
export const copyAgentKey = (client: string) => AppService.AgentCopyKey(client)
export const setAgentEnabled = (client: string, enabled: boolean) =>
  AppService.AgentSetEnabled(client, enabled)

export function agentError(error: unknown) {
  const code = String(error instanceof Error ? error.message : error)
    .split("\n", 1)[0]
    ?.replace("agent: ", "")
  switch (code) {
    case "config_changed":
    case "client_upgrade_required":
    case "invalid_config":
    case "auth_conflict":
    case "profile_conflict":
    case "key_disabled":
    case "desktop_profile_active":
    case "no_models":
    case "invalid_model_selection":
    case "invalid_model":
    case "models_unavailable":
    case "catalog_changed":
    case "catalog_conflict":
    case "catalog_too_large":
    case "write_failed":
    case "read_failed":
    case "unsafe_config":
    case "config_locked":
    case "backup_failed":
    case "backup_unavailable":
    case "rollback_failed":
      return code
    default:
      return "failed"
  }
}
export function agentTitle(client: string) {
  switch (client) {
    case "codex":
      return "agents.codex"
    case "claude":
      return "agents.claude"
    case "claude-desktop":
      return "agents.claude-desktop"
    case "opencode":
      return "agents.opencode"
    case "harness-web":
      return "agents.harnessWeb"
    case "harness-desktop":
      return "agents.harnessDesktop"
    case "kilocode":
      return "agents.kilocode"
    case "qwen-code":
      return "agents.qwenCode"
    case "mimo-code":
      return "agents.mimoCode"
    default:
      return "agents.title"
  }
}
