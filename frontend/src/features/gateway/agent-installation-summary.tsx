import { useTranslation } from "react-i18next"
import type { AgentInstallSnapshot } from "../../services/agent-installation"
import { agentInstallationTargets, type AgentInstallTarget } from "./agent-installation-targets"
import { agentInstallationStatus } from "./agent-installation-status"

export function AgentInstallationSummary({
  target,
  snapshot,
  unavailable,
}: {
  target: AgentInstallTarget
  snapshot?: AgentInstallSnapshot
  unavailable?: boolean
}) {
  const { t } = useTranslation()
  const item = snapshot?.items?.find((item) => item.id === target)
  return (
    <dl className="space-y-1.5 text-xs">
      <dt className="sr-only">{t(agentInstallationTargets[target])}</dt>
      <dd
        className={
          item?.detection.state === "installed"
            ? "text-emerald-700 dark:text-emerald-400"
            : "text-muted-foreground"
        }
      >
        {t(
          snapshot
            ? agentInstallationStatus(item)
            : unavailable
              ? "agents.install.unknown"
              : "agents.install.checking",
        )}
      </dd>
    </dl>
  )
}
