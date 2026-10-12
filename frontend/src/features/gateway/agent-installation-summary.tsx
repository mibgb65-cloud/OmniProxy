import { useTranslation } from "react-i18next"
import type { AgentInstallSnapshot } from "../../services/agent-installation"
import type { AgentClientId } from "./agent-clients"
import { agentInstallationTargets, clientInstallationTargets } from "./agent-installation-targets"
import { agentInstallationStatus } from "./agent-installation-status"

export function AgentInstallationSummary({
  client,
  snapshot,
  unavailable,
}: {
  client: AgentClientId
  snapshot?: AgentInstallSnapshot
  unavailable?: boolean
}) {
  const { t } = useTranslation()
  return (
    <dl className="space-y-1.5 text-xs">
      {clientInstallationTargets[client].map((id) => {
        const item = snapshot?.items?.find((item) => item.id === id)
        return (
          <div key={id} className="flex flex-wrap items-center justify-between gap-2">
            <dt className="text-muted-foreground">{t(agentInstallationTargets[id])}</dt>
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
          </div>
        )
      })}
    </dl>
  )
}
