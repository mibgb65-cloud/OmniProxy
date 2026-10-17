import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type { Snapshot, Item } from "../../bindings/omniproxy/internal/application/agentinstall"

export type AgentInstallSnapshot = Snapshot
export type AgentInstallation = Item
export type { UpdateCheck as AgentUpdateCheck } from "../../bindings/omniproxy/internal/infrastructure/agentinstall"
export const getAgentInstallations = () => AppService.AgentInstallations()
export const refreshAgentInstallations = () => AppService.AgentInstallationsRefresh()
export const installAgent = (id: string) => AppService.AgentInstall(id)
export const checkAgentUpdate = (id: string) => AppService.AgentInstallationCheckUpdate(id)
export const updateAgent = (id: string) => AppService.AgentInstallationUpdate(id)
export const openAgentUpdate = (id: string) => AppService.AgentInstallationOpenUpdate(id)
export const openAgentInstallationOfficial = (id: string) =>
  AppService.AgentInstallationOpenOfficial(id)
export const openClaudeCode = () => AppService.AgentOpenClaudeCode()
export const openAgentClient = (id: string) => AppService.AgentOpenClient(id)

export { agentInstallationError } from "./agent-installation-error"
