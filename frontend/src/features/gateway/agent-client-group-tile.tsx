import { useId } from "react"
import { useTranslation } from "react-i18next"
import type { AgentClient } from "../../services/agents"
import type { AgentInstallSnapshot } from "../../services/agent-installation"
import type { AgentClientId } from "./agent-clients"
import { AgentBrandIcon } from "./agent-brand-icon"
import { AgentClientTile } from "./agent-client-tile"

export type AgentClientGroupProps = {
  items: AgentClient[]
  pending: boolean
  installations?: AgentInstallSnapshot
  installationsUnavailable?: boolean
}

export function AgentClientGroupTile({
  items,
  pending,
  installations,
  installationsUnavailable,
  title,
  description,
  clients,
}: AgentClientGroupProps & {
  title: "agents.claudeGroup" | "agents.harness"
  description: "agents.claudeGroupDescription" | "agents.harnessDescription"
  clients: readonly [AgentClientId, ...AgentClientId[]]
}) {
  const { t } = useTranslation()
  const titleId = useId()
  return (
    <section
      aria-labelledby={titleId}
      className="bg-card flex min-w-0 flex-col gap-5 rounded-xl border p-6"
    >
      <header className="flex items-center gap-4">
        <AgentBrandIcon client={clients[0]} />
        <div className="min-w-0 space-y-2">
          <h3 id={titleId} className="text-base font-semibold">
            {t(title)}
          </h3>
          <p className="text-muted-foreground text-sm leading-6">{t(description)}</p>
        </div>
      </header>
      <div className="grid gap-3">
        {clients.map((client) => (
          <AgentClientTile
            key={client}
            client={client}
            item={items.find((item) => item.client === client)}
            pending={pending}
            installations={installations}
            installationsUnavailable={installationsUnavailable}
            compact
          />
        ))}
      </div>
    </section>
  )
}
