import { useQuery } from "@tanstack/react-query"
import { useState } from "react"
import { Boxes, RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { ScrollRegion } from "../../components/common/scroll-region"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { Input } from "../../components/ui/input"
import { agentError, getConfiguredAgentModels, type AgentClient } from "../../services/agents"

export function AgentConnectedModels({ item }: { item: AgentClient }) {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const data = useQuery({
    queryKey: ["agent-configured-models", item.client, item.revision, item.canRestore],
    queryFn: () => getConfiguredAgentModels(item.client),
    enabled: item.canRestore,
  })
  const models = item.canRestore && !data.isError ? (data.data?.models ?? []) : []
  const visible = models.filter((model) =>
    `${model.name} ${model.id}`.toLowerCase().includes(query.trim().toLowerCase()),
  )
  return (
    <Card role="region" aria-labelledby="agent-connected-models-title" className="min-w-0 gap-4">
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <h3 id="agent-connected-models-title" className="text-base font-semibold">
            {t("agents.connectedModels")}
          </h3>
          <span className="bg-muted text-muted-foreground rounded-md px-2 py-0.5 text-xs">
            {models.length}
          </span>
        </div>
        <Button
          variant="outline"
          size="sm"
          disabled={!item.canRestore || data.isFetching}
          onClick={() => void data.refetch()}
        >
          <RefreshCw
            aria-hidden="true"
            className={data.isFetching ? "motion-safe:animate-spin" : ""}
          />
          {t("agents.refreshModels")}
        </Button>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-muted-foreground text-xs leading-5">{t("agents.connectedModelsHint")}</p>
        {item.canRestore && !item.configured ? (
          <p className="text-muted-foreground text-xs">{t("agents.modelsKeyDisabled")}</p>
        ) : null}
        {data.isError ? (
          <p role="alert" className="text-destructive text-sm">
            {t(`agents.error.${agentError(data.error)}`)}
          </p>
        ) : item.canRestore && data.isPending ? (
          <p className="text-muted-foreground py-8 text-sm">{t("agents.loading")}</p>
        ) : models.length ? (
          <>
            <Input
              type="search"
              value={query}
              aria-label={t("agents.searchConnectedModels")}
              placeholder={t("agents.searchConnectedModels")}
              onChange={(event) => setQuery(event.target.value)}
            />
            <ScrollRegion
              as="section"
              ariaLabel={t("agents.connectedModels")}
              className="max-h-[min(34rem,60dvh)]"
              viewportClassName="pr-2"
            >
              <ul className="divide-y">
                {visible.map((model) => (
                  <li key={model.id} className="min-w-0 space-y-2 py-3 first:pt-0">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <span className="min-w-0 text-sm font-medium wrap-anywhere">
                        {model.name || model.id}
                      </span>
                      {model.id === data.data?.defaultModel ? (
                        <span className="bg-muted text-muted-foreground rounded-md border px-2 py-0.5 text-xs">
                          {t("agents.defaultModel")}
                        </span>
                      ) : null}
                    </div>
                    <p className="text-muted-foreground font-mono text-xs wrap-anywhere">
                      {model.id}
                    </p>
                  </li>
                ))}
              </ul>
              {!visible.length ? (
                <p className="text-muted-foreground py-6 text-sm">{t("agents.noMatchingModels")}</p>
              ) : null}
            </ScrollRegion>
          </>
        ) : (
          <div className="text-muted-foreground flex min-h-56 flex-col items-center justify-center gap-3 rounded-lg border border-dashed p-5 text-center">
            <Boxes aria-hidden="true" className="size-8 opacity-60" />
            <p className="text-sm font-medium">{t("agents.noConnectedModels")}</p>
            <p className="max-w-xs text-xs leading-5">{t("agents.noConnectedModelsHint")}</p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
