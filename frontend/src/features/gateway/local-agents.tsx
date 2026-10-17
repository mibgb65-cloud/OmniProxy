import { useQuery } from "@tanstack/react-query"
import { useEffect, useRef } from "react"
import { useTranslation } from "react-i18next"
import { listAgentClients } from "../../services/agents"
import { AgentClientDetail } from "./agent-client-detail"
import { AgentClientGroupTile } from "./agent-client-group-tile"
import { agentClientGroups, type AgentClientEntry } from "./agent-client-groups"
import { type AgentClientId } from "./agent-clients"
import { useAgentInstallations } from "./use-agent-installations"
import "./agent-client-grid.css"

export function LocalAgents({
  models,
  selectedClient,
  active,
}: {
  models: string[]
  selectedClient?: AgentClientId
  active: boolean
}) {
  const { t } = useTranslation()
  const data = useQuery({ queryKey: ["agent-clients"], queryFn: listAgentClients })
  const installations = useAgentInstallations(active)
  const rootRef = useRef<HTMLElement>(null)
  const previousClient = useRef<AgentClientId | undefined>(undefined)
  const openedEntry = useRef<AgentClientEntry | undefined>(undefined)
  // 父页签在布局阶段重置滚动，随后恢复软件入口焦点，让浏览器将它滚入可见区域。
  // 用软件目标区分共用详情的 CLI 和 Desktop，保持现有详情地址。
  useEffect(() => {
    if (!active) return
    if (selectedClient)
      rootRef.current
        ?.querySelector<HTMLElement>("[data-agent-title]")
        ?.focus({ preventScroll: true })
    else if (previousClient.current) {
      const selector =
        openedEntry.current?.client === previousClient.current
          ? `[data-agent-entry="${openedEntry.current.target}"]`
          : `[data-agent-client="${previousClient.current}"]`
      rootRef.current?.querySelector<HTMLElement>(selector)?.focus()
    }
    previousClient.current = selectedClient
  }, [selectedClient, active])
  const items = data.isError ? [] : (data.data ?? [])
  return (
    <section ref={rootRef} className="space-y-5" aria-label={t("agents.title")}>
      {data.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("agents.loadFailed")}
        </p>
      ) : null}
      {selectedClient ? (
        <AgentClientDetail
          client={selectedClient}
          item={items.find((item) => item.client === selectedClient)}
          pending={data.isPending}
          models={models}
          installations={installations}
        />
      ) : (
        <>
          <div className="space-y-2">
            <h2 className="text-base font-semibold">{t("agents.title")}</h2>
            <p className="text-muted-foreground text-sm">{t("agents.selectClient")}</p>
          </div>
          <div className="agent-client-grid grid min-w-0 gap-4">
            {agentClientGroups.map((group) => (
              <AgentClientGroupTile
                key={group.title}
                group={group}
                items={items}
                pending={data.isPending}
                installations={installations.data.isError ? undefined : installations.data.data}
                installationsUnavailable={installations.data.isError}
                onOpen={(entry) => {
                  openedEntry.current = entry
                }}
              />
            ))}
          </div>
        </>
      )}
    </section>
  )
}
