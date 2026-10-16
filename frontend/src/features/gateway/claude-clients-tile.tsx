import { useId } from "react"
import { useTranslation } from "react-i18next"
import type { AgentClient } from "../../services/agents"
import type { AgentInstallSnapshot } from "../../services/agent-installation"
import { AgentBrandIcon } from "./agent-brand-icon"
import { AgentClientTile } from "./agent-client-tile"

export function ClaudeClientsTile({
  items,
  pending,
  installations,
  installationsUnavailable,
}: {
  items: AgentClient[]
  pending: boolean
  installations?: AgentInstallSnapshot
  installationsUnavailable?: boolean
}) {
  const { t } = useTranslation()
  const titleId = useId()
  return (
    <section
      aria-labelledby={titleId}
      className="bg-card flex min-w-0 flex-col gap-5 rounded-xl border p-6"
    >
      <header className="flex items-center gap-4">
        <AgentBrandIcon client="claude" />
        <div className="min-w-0 space-y-2">
          <h3 id={titleId} className="text-base font-semibold">
            {t("agents.claudeGroup")}
          </h3>
          <p className="text-muted-foreground text-sm leading-6">
            {t("agents.claudeGroupDescription")}
          </p>
        </div>
      </header>
      <div className="grid gap-3">
        {(["claude", "claude-desktop"] as const).map((client) => (
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
