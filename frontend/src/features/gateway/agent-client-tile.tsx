import { Link } from "@tanstack/react-router"
import { ArrowUpRight } from "lucide-react"
import { useTranslation } from "react-i18next"
import type { AgentClient } from "../../services/agents"
import { AgentBrandIcon } from "./agent-brand-icon"
import { agentClients, agentStatusKey, type AgentClientId } from "./agent-clients"
import { AgentInstallationSummary } from "./agent-installation-summary"
import type { AgentInstallSnapshot } from "../../services/agent-installation"

export function AgentClientTile({
  client,
  item,
  pending,
  installations,
  installationsUnavailable,
  compact = false,
}: {
  client: AgentClientId
  item?: AgentClient
  pending: boolean
  installations?: AgentInstallSnapshot
  installationsUnavailable?: boolean
  compact?: boolean
}) {
  const { t } = useTranslation()
  const agent = agentClients[client]
  const Title = compact ? "h4" : "h3"
  return (
    <Link
      to="/gateway"
      search={{ tab: "clients", client }}
      aria-label={t("agents.openClient", { client: t(agent.title) })}
      data-agent-client={client}
      className={`group bg-card hover:bg-accent/40 focus-visible:ring-ring flex min-w-0 flex-col rounded-xl border transition-colors outline-none focus-visible:ring-2 focus-visible:ring-offset-2 ${compact ? "gap-2 p-3" : "min-h-56 gap-5 p-6"}`}
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        {compact ? (
          <Title className="text-sm font-medium">{t(agent.title)}</Title>
        ) : (
          <AgentBrandIcon client={client} />
        )}
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
      {!compact ? (
        <div className="min-w-0 space-y-2">
          <Title className="text-base font-semibold">{t(agent.title)}</Title>
          <p className="text-muted-foreground text-sm leading-6">{t(agent.description)}</p>
        </div>
      ) : null}
      <AgentInstallationSummary
        client={client}
        snapshot={installations}
        unavailable={installationsUnavailable}
        compact={compact}
      />
      <div
        className={`text-muted-foreground group-hover:text-foreground mt-auto flex items-center justify-between gap-3 ${compact ? "text-xs" : "text-sm"}`}
      >
        <span>{t("agents.open")}</span>
        <ArrowUpRight aria-hidden="true" className="size-4" />
      </div>
    </Link>
  )
}
