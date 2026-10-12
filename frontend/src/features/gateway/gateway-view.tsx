import { Activity, Code, FlaskConical, Terminal } from "lucide-react"
import { useLayoutEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../../components/ui/tabs"
import { GatewayClientCard } from "./gateway-client-card"
import { GatewayConfigCard } from "./gateway-config-card"
import { GatewayConnectionCard } from "./gateway-connection-card"
import { GatewayHeader } from "./gateway-header"
import { GatewayReadinessCard } from "./gateway-readiness-card"
import { GatewayRelatedLinks } from "./gateway-related-links"
import { GatewayStatusCard } from "./gateway-status-card"
import { gatewayTabs, isGatewayTab, type GatewayTab } from "./gateway-tabs"
import { LocalAgents } from "./local-agents"
import type { AgentClientId } from "./agent-clients"
import { useGatewayPreparation } from "./use-gateway-preparation"

const tabIcons = { overview: Activity, clients: Terminal, manual: Code, diagnostics: FlaskConical }
const tabLabels = {
  overview: "gateway.tabs.overview",
  clients: "gateway.tabs.clients",
  manual: "gateway.tabs.manual",
  diagnostics: "gateway.tabs.diagnostics",
} as const

export function GatewayView({
  tab,
  client,
  onTabChange,
}: {
  tab: GatewayTab
  client?: AgentClientId
  onTabChange: (tab: GatewayTab) => void
}) {
  const { t } = useTranslation()
  const data = useGatewayPreparation()
  const pageRef = useRef<HTMLDivElement>(null)
  const [visited, setVisited] = useState<GatewayTab[]>([tab])
  // 路由前进／后退也会切换页签；首次访问后保留组件，避免卸载丢失输入与异步反馈。
  if (!visited.includes(tab)) setVisited([...visited, tab])

  useLayoutEffect(() => {
    const viewport = pageRef.current?.closest<HTMLElement>(".scroll-region__viewport")
    if (viewport) viewport.scrollTop = 0
  }, [tab, client])

  return (
    <Tabs
      ref={pageRef}
      value={tab}
      activationMode="manual"
      onValueChange={(value) => {
        if (isGatewayTab(value)) onTabChange(value)
      }}
      className="w-full min-w-0 space-y-5 pb-6"
    >
      <div className="bg-background/95 sticky top-0 z-20 space-y-5 pt-1 backdrop-blur-sm">
        <GatewayHeader data={data} />
        <TabsList aria-label={t("gateway.tabs.title")}>
          {gatewayTabs.map((value) => {
            const Icon = tabIcons[value]
            return (
              <TabsTrigger key={value} value={value}>
                <Icon aria-hidden="true" />
                {t(tabLabels[value])}
              </TabsTrigger>
            )
          })}
        </TabsList>
      </div>
      <TabsContent
        value="overview"
        forceMount={visited.includes("overview") ? true : undefined}
        hidden={tab !== "overview"}
      >
        <p className="text-muted-foreground text-sm">{t("gateway.tabs.overviewHint")}</p>
        <div className="grid min-w-0 gap-5 xl:grid-cols-2">
          <GatewayStatusCard data={data} />
          <GatewayConfigCard data={data} />
        </div>
        <GatewayRelatedLinks />
      </TabsContent>
      <TabsContent
        value="clients"
        forceMount={visited.includes("clients") ? true : undefined}
        hidden={tab !== "clients"}
      >
        <LocalAgents
          selectedClient={client}
          active={tab === "clients"}
          models={data.gateway.info.data?.models ?? []}
          running={data.gateway.isRunning}
        />
      </TabsContent>
      <TabsContent
        value="manual"
        forceMount={visited.includes("manual") ? true : undefined}
        hidden={tab !== "manual"}
      >
        <p className="text-muted-foreground text-sm">{t("gateway.tabs.manualHint")}</p>
        <GatewayConnectionCard data={data} />
        <div className="grid min-w-0 gap-5 xl:grid-cols-2">
          <GatewayClientCard client="codex" data={data} />
          <GatewayClientCard client="claude" data={data} />
        </div>
      </TabsContent>
      <TabsContent
        value="diagnostics"
        forceMount={visited.includes("diagnostics") ? true : undefined}
        hidden={tab !== "diagnostics"}
      >
        <p className="text-muted-foreground text-sm">{t("gateway.tabs.diagnosticsHint")}</p>
        <GatewayReadinessCard data={data} />
        <GatewayRelatedLinks diagnostics />
      </TabsContent>
    </Tabs>
  )
}
