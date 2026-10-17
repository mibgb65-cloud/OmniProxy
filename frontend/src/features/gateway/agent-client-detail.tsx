import { Link } from "@tanstack/react-router"
import { ArrowLeft } from "lucide-react"
import { useTranslation } from "react-i18next"
import type { AgentClient } from "../../services/agents"
import { AgentBrandIcon } from "./agent-brand-icon"
import { AgentDetailWorkspace } from "./agent-detail-workspace"
import { AgentInstallationCard } from "./agent-installation-card"
import { agentClients, agentStatusKey, type AgentClientId } from "./agent-clients"
import type { AgentInstallations } from "./use-agent-installations"

export function AgentClientDetail({
  client,
  item,
  pending,
  models,
  running,
  installations,
}: {
  client: AgentClientId
  item?: AgentClient
  pending: boolean
  models: string[]
  running: boolean
  installations: AgentInstallations
}) {
  const { t } = useTranslation()
  const agent = agentClients[client]
  return (
    <div className="space-y-6">
      <Link
        to="/gateway"
        search={{ tab: "clients", client: undefined }}
        className="text-muted-foreground hover:text-foreground focus-visible:ring-ring inline-flex items-center gap-2 rounded-md py-1 text-sm outline-none focus-visible:ring-2"
      >
        <ArrowLeft aria-hidden="true" className="size-4" />
        {t("agents.back")}
      </Link>
      <div className="flex flex-wrap items-center gap-4">
        <AgentBrandIcon client={client} />
        <div className="min-w-0 space-y-2">
          <div className="flex flex-wrap items-center gap-3">
            <h2 data-agent-title tabIndex={-1} className="text-xl font-semibold outline-none">
              {t(agent.title)}
            </h2>
            <span className="bg-muted text-muted-foreground rounded-full border px-2.5 py-1 text-xs">
              {t(pending ? "agents.loading" : agentStatusKey(item))}
            </span>
          </div>
          <p className="text-muted-foreground text-sm">{t(agent.description)}</p>
        </div>
      </div>
      {client === "opencode" || client.startsWith("harness-") ? (
        <p className="text-muted-foreground text-sm leading-6">
          {t(client === "opencode" ? "agents.opencodeHint" : "agents.harnessHint")}
        </p>
      ) : null}
      {client === "kilocode" || client === "qwen-code" || client === "mimo-code" ? (
        <p className="text-muted-foreground text-sm leading-6">
          {t(
            client === "qwen-code"
              ? "agents.qwenCodeHint"
              : client === "mimo-code"
                ? "agents.mimoCodeHint"
                : "agents.codingEngineHint",
          )}
        </p>
      ) : null}
      {item ? (
        <AgentDetailWorkspace
          key={client}
          client={client}
          item={item}
          models={models}
          running={running}
          installations={installations}
        />
      ) : (
        <>
          <AgentInstallationCard client={client} state={installations} />
          <p role={pending ? "status" : "alert"} className="text-muted-foreground text-sm">
            {t(pending ? "agents.loading" : "agents.unavailable")}
          </p>
        </>
      )}
    </div>
  )
}
