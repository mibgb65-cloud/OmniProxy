import { Loader2, Sparkles } from "lucide-react"
import { useId } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { AgentClient } from "../../services/agents"
import type { AgentInstallSnapshot } from "../../services/agent-installation"
import type { AgentClientEntry, AgentClientGroup } from "./agent-client-groups"
import { agentClients } from "./agent-clients"
import { AgentBrandIcon } from "./agent-brand-icon"
import { AgentClientTile } from "./agent-client-tile"
import { useAgentGroupConfig } from "./use-agent-group-config"

type AgentClientGroupProps = {
  items: AgentClient[]
  pending: boolean
  models: string[]
  installations?: AgentInstallSnapshot
  installationsUnavailable?: boolean
  group: AgentClientGroup
  onOpen: (entry: AgentClientEntry) => void
}

export function AgentClientGroupTile({
  items,
  pending,
  models,
  installations,
  installationsUnavailable,
  group,
  onOpen,
}: AgentClientGroupProps) {
  const { t } = useTranslation()
  const titleId = useId()
  const groupTitle = t(group.title)
  const config = useAgentGroupConfig(
    group.entries.map((entry) => entry.client),
    items,
  )
  const noModels = !models.length
  return (
    <section
      aria-labelledby={titleId}
      className="bg-card flex min-w-0 flex-col gap-5 rounded-xl border p-6"
    >
      <header className="flex items-center gap-4">
        <AgentBrandIcon client={group.entries[0].client} />
        <div className="min-w-0 space-y-2">
          <h3 id={titleId} className="text-base font-semibold">
            {groupTitle}
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
      {/* 卡片级一键配置：直接为该分组全部客户端写入完整默认目录，不必进入详情手动挑选模型。 */}
      <footer className="mt-auto space-y-2 border-t pt-4">
        <Button
          variant="outline"
          size="sm"
          className="w-full"
          aria-label={t("agents.groupConfigureFor", { client: groupTitle })}
          disabled={pending || config.busy || noModels}
          onClick={() => void config.run()}
        >
          {config.busy ? (
            <Loader2 className="motion-safe:animate-spin" aria-hidden="true" />
          ) : (
            <Sparkles aria-hidden="true" />
          )}
          {t("agents.configure")}
        </Button>
        {noModels ? (
          <p className="text-muted-foreground text-xs leading-5">
            {t("agents.groupConfigureUnavailable")}
          </p>
        ) : null}
        {config.busy ? (
          <p role="status" className="text-muted-foreground text-xs leading-5">
            {t("agents.groupConfiguring", { done: config.done, total: config.total })}
          </p>
        ) : null}
        {!config.busy && config.finished !== null ? (
          <div className="space-y-1 text-xs leading-5">
            <p
              role="status"
              className={
                config.failures.length
                  ? "text-amber-700 dark:text-amber-400"
                  : "text-emerald-700 dark:text-emerald-400"
              }
            >
              {t("agents.groupConfigured", { done: config.finished, total: config.total })}
            </p>
            {config.failures.length ? (
              <ul className="text-muted-foreground space-y-1">
                {config.failures.map((failure) => (
                  <li key={failure.client}>
                    {t("agents.groupConfigureFailedItem", {
                      client: t(agentClients[failure.client].title),
                      reason: t(`agents.error.${failure.error}`),
                    })}
                  </li>
                ))}
              </ul>
            ) : null}
          </div>
        ) : null}
      </footer>
    </section>
  )
}
