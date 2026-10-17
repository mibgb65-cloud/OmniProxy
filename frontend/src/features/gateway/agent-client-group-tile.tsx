import { useId } from "react"
import { useTranslation } from "react-i18next"
import type { AgentClient } from "../../services/agents"
import type { AgentInstallSnapshot } from "../../services/agent-installation"
import type { AgentClientEntry, AgentClientGroup } from "./agent-client-groups"
import { AgentBrandIcon } from "./agent-brand-icon"
import { AgentClientTile } from "./agent-client-tile"

type AgentClientGroupProps = {
  items: AgentClient[]
  pending: boolean
  installations?: AgentInstallSnapshot
  installationsUnavailable?: boolean
  group: AgentClientGroup
  onOpen: (entry: AgentClientEntry) => void
}

export function AgentClientGroupTile({
  items,
  pending,
  installations,
  installationsUnavailable,
  group,
  onOpen,
}: AgentClientGroupProps) {
  const { t } = useTranslation()
  const titleId = useId()
  return (
    <section
      aria-labelledby={titleId}
      className="bg-card flex min-w-0 flex-col gap-5 rounded-xl border p-6"
    >
      <header className="flex items-center gap-4">
        <AgentBrandIcon client={group.entries[0].client} />
        <div className="min-w-0 space-y-2">
          <h3 id={titleId} className="text-base font-semibold">
            {t(group.title)}
          </h3>
          <p className="text-muted-foreground text-sm leading-6">{t(group.description)}</p>
        </div>
      </header>
      <div className="grid gap-3">
        {group.entries.map((entry) => (
          <AgentClientTile
            key={entry.target}
            entry={entry}
            item={items.find((item) => item.client === entry.client)}
            pending={pending}
            installations={installations}
            installationsUnavailable={installationsUnavailable}
            onOpen={onOpen}
          />
        ))}
      </div>
    </section>
  )
}
