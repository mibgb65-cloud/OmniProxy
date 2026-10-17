import { Link } from "@tanstack/react-router"
import { ArrowUpRight } from "lucide-react"
import { useTranslation } from "react-i18next"
import type { AgentClient } from "../../services/agents"
import { agentStatusKey } from "./agent-clients"
import type { AgentClientEntry } from "./agent-client-groups"
import { agentInstallationTargets } from "./agent-installation-targets"
import { AgentInstallationSummary } from "./agent-installation-summary"
import type { AgentInstallSnapshot } from "../../services/agent-installation"

export function AgentClientTile({
  entry,
  item,
  pending,
  installations,
  installationsUnavailable,
  onOpen,
}: {
  entry: AgentClientEntry
  item?: AgentClient
  pending: boolean
  installations?: AgentInstallSnapshot
  installationsUnavailable?: boolean
  onOpen: (entry: AgentClientEntry) => void
}) {
  const { t } = useTranslation()
  const title = t(entry.title ?? agentInstallationTargets[entry.target])
  return (
    <Link
      to="/gateway"
      search={{ tab: "clients", client: entry.client }}
      aria-label={t("agents.openClient", { client: title })}
      data-agent-client={entry.client}
      data-agent-entry={entry.target}
      onClick={() => onOpen(entry)}
      className="group bg-card hover:bg-accent/40 focus-visible:ring-ring flex min-w-0 flex-col gap-2 rounded-xl border p-3 transition-colors outline-none focus-visible:ring-2 focus-visible:ring-offset-2"
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <h4 className="text-sm font-medium">{title}</h4>
        <span
          className={`flex items-center gap-1.5 rounded-full border px-2 py-1 text-xs ${item?.configured && !item.errorCode ? "border-emerald-500/20 text-emerald-700 dark:text-emerald-400" : "text-muted-foreground"}`}
        >
          <span
            aria-hidden="true"
            className={`size-1.5 shrink-0 rounded-full ${item?.configured && !item.errorCode ? "bg-emerald-500" : "bg-muted-foreground/60"}`}
          />
          {t(pending ? "agents.loading" : agentStatusKey(item))}
        </span>
      </div>
      <AgentInstallationSummary
        target={entry.target}
        snapshot={installations}
        unavailable={installationsUnavailable}
      />
      <div className="text-muted-foreground group-hover:text-foreground mt-auto flex items-center justify-between gap-3 text-xs">
        <span>{t("agents.open")}</span>
        <ArrowUpRight aria-hidden="true" className="size-4" />
      </div>
    </Link>
  )
}
