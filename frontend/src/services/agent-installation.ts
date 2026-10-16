import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type { Snapshot, Item } from "../../bindings/omniproxy/internal/application/agentinstall"

export type AgentInstallSnapshot = Snapshot
export type AgentInstallation = Item
export const getAgentInstallations = () => AppService.AgentInstallations()
export const refreshAgentInstallations = () => AppService.AgentInstallationsRefresh()
export const installAgent = (id: string) => AppService.AgentInstall(id)
export const openAgentInstallationOfficial = (id: string) =>
  AppService.AgentInstallationOpenOfficial(id)
export const openClaudeCode = () => AppService.AgentOpenClaudeCode()
export const openAgentClient = (id: string) => AppService.AgentOpenClient(id)

export function agentInstallationError(error: unknown) {
  const code = String(error instanceof Error ? error.message : error).replace("agentinstall: ", "")
  switch (code) {
    case "installer_unavailable":
    case "detection_failed":
    case "version_unavailable":
    case "install_failed":
    case "install_interrupted":
    case "verification_failed":
    case "busy":
    case "unsupported_platform":
    case "official_unavailable":
    case "launch_failed":
    case "activation_required":
      return code
    default:
      return "failed"
  }
}
