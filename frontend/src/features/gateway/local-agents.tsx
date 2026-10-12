import { useQuery } from "@tanstack/react-query"
import { useLayoutEffect, useRef } from "react"
import { useTranslation } from "react-i18next"
import { listAgentClients } from "../../services/agents"
import { AgentClientDetail } from "./agent-client-detail"
import { AgentClientTile } from "./agent-client-tile"
import { agentClientIds, type AgentClientId } from "./agent-clients"
import { useAgentInstallations } from "./use-agent-installations"

export function LocalAgents({
  models,
  running,
  selectedClient,
  active,
}: {
  models: string[]
  running: boolean
  selectedClient?: AgentClientId
  active: boolean
}) {
  const { t } = useTranslation()
  const data = useQuery({ queryKey: ["agent-clients"], queryFn: listAgentClients })
  const installations = useAgentInstallations(active)
  const rootRef = useRef<HTMLElement>(null)
  const previousClient = useRef<AgentClientId | undefined>(undefined)
  // 详情导航后聚焦标题；返回目录时恢复卡片焦点，隐藏选项卡不抢焦点。
  useLayoutEffect(() => {
    if (!active) return
    if (selectedClient)
      rootRef.current
        ?.querySelector<HTMLElement>("[data-agent-title]")
        ?.focus({ preventScroll: true })
    else if (previousClient.current)
      rootRef.current
        ?.querySelector<HTMLElement>(`[data-agent-client="${previousClient.current}"]`)
        ?.focus({ preventScroll: true })
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
          running={running}
          installations={installations}
        />
      ) : (
        <>
          <div className="space-y-2">
            <h2 className="text-base font-semibold">{t("agents.title")}</h2>
            <p className="text-muted-foreground text-sm">{t("agents.selectClient")}</p>
          </div>
          <div className="grid min-w-0 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {agentClientIds.map((client) => (
              <AgentClientTile
                key={client}
                client={client}
                item={items.find((item) => item.client === client)}
                pending={data.isPending}
                installations={installations.data.isError ? undefined : installations.data.data}
                installationsUnavailable={installations.data.isError}
              />
            ))}
          </div>
        </>
      )}
    </section>
  )
}
