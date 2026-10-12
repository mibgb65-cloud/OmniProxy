import type { AgentInstallation } from "../../services/agent-installation"

export function agentInstallationStatus(item?: AgentInstallation) {
  if (!item) return "agents.install.unknown"
  if (item.installState === "checking") return "agents.install.checking"
  if (item.installState === "installing") {
    if (item.progress?.phase === "downloading") return "agents.install.downloading"
    if (item.progress?.phase === "installing") return "agents.install.installingStage"
    return "agents.install.installing"
  }
  if (item.installState === "verifying") return "agents.install.verifying"
  if (item.detection.state === "installed") return "agents.install.installed"
  if (item.detection.state === "not_installed") return "agents.install.notInstalled"
  return "agents.install.unknown"
}
