import { useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { Input } from "../../components/ui/input"
import { DefaultModelRouteRow } from "./default-model-route-row"
import { DefaultProviderRoutes } from "./default-provider-routes"
import { ModelProviderViewToggle, type ModelProviderView } from "./model-provider-view-toggle"
import type { RoutingData } from "./use-routing"
import type { DefaultModelRoute } from "../../services/routing"
import { ProviderPriorityEditor } from "./provider-priority-editor"
import { useLastValue } from "../../lib/use-last-value"
import "./default-model-route-grid.css"

export function DefaultModelRoutes({ data }: { data: RoutingData }) {
  const { t } = useTranslation()
  const [search, setSearch] = useState("")
  const [view, setView] = useState<ModelProviderView>("model")
  const [priorityTarget, setPriorityTarget] = useState<DefaultModelRoute | null>(null)
  // 关闭动画期间仍需要这条路由可渲染，保留最后一次打开的目标。
  const priority = useLastValue(priorityTarget)
  const returnFocus = useRef<HTMLElement | null>(null)
  const cardRef = useRef<HTMLDivElement>(null)
  function configurePriority(item: DefaultModelRoute) {
    returnFocus.current =
      document.activeElement instanceof HTMLElement ? document.activeElement : null
    setPriorityTarget(item)
  }
  const items = data.defaults.data ?? []
  const visible = items.filter((item) =>
    [
      item.model,
      item.name,
      item.preview.providerName,
      ...(item.providers ?? []).map((p) => p.name),
    ].some((value) => value.toLowerCase().includes(search.toLowerCase())),
  )
  const busy =
    data.preference.isPending ||
    data.priority.isPending ||
    data.create.isPending ||
    data.update.isPending ||
    data.toggle.isPending ||
    data.remove.isPending ||
    data.defaults.isFetching
  return (
    <Card ref={cardRef} className="min-w-0 gap-4" aria-label={t("routing.defaults.title")}>
      <CardHeader className="flex flex-wrap justify-between gap-3">
        <div className="min-w-0 flex-1 basis-80 space-y-2">
          <h2 className="text-base font-semibold">{t("routing.defaults.title")}</h2>
          <p className="text-muted-foreground text-sm">{t("routing.defaults.description")}</p>
        </div>
        <div className="ml-auto flex max-w-full flex-col items-end gap-2">
          <span className="text-muted-foreground text-xs">
            {t("routing.defaults.count", { count: items.length })}
          </span>
          <ModelProviderViewToggle value={view} onChange={setView} />
        </div>
      </CardHeader>
      <CardContent className="min-w-0 space-y-4">
        {data.defaults.isPending && !data.catalog.isError ? (
          <p role="status" className="text-muted-foreground text-sm">
            {t("routing.defaults.loading")}
          </p>
        ) : null}
        {data.defaults.isError || data.catalog.isError ? (
          <p role="alert" className="text-destructive text-sm">
            {t("routing.defaults.loadFailed")}
          </p>
        ) : null}
        {data.preference.isError ? (
          <p role="alert" className="text-destructive text-sm">
            {t("routing.defaults.saveFailed")}
          </p>
        ) : null}
        {data.preference.isPending || data.preference.isSuccess ? (
          <p role="status" className="text-muted-foreground text-xs">
            {t(data.preference.isPending ? "routing.defaults.saving" : "routing.defaults.saved", {
              model: data.preference.variables?.model,
            })}
          </p>
        ) : null}
        {data.priority.isSuccess ? (
          <p role="status" className="text-muted-foreground text-xs">
            {t("routing.priority.saved", { model: data.priority.variables?.model })}
          </p>
        ) : null}
        {items.length ? (
          <>
            <Input
              aria-label={t("routing.defaults.search")}
              placeholder={t("routing.searchPlaceholder")}
              maxLength={128}
              value={search}
              onChange={(event) => setSearch(event.target.value)}
            />
            {view === "provider" ? (
              <DefaultProviderRoutes
                items={items}
                search={search}
                disabled={busy}
                onSelect={(model, providerId) => data.preference.mutate({ model, providerId })}
                onConfigurePriority={configurePriority}
              />
            ) : (
              <ul
                className="default-model-route-grid grid min-w-0 gap-3"
                aria-label={t("routing.defaults.title")}
              >
                {visible.map((item) => (
                  <DefaultModelRouteRow
                    key={item.model}
                    item={item}
                    disabled={busy}
                    onSelect={(providerId) =>
                      data.preference.mutate({ model: item.model, providerId })
                    }
                    onConfigurePriority={() => configurePriority(item)}
                  />
                ))}
              </ul>
            )}
            {view === "model" && !visible.length ? (
              <p className="text-muted-foreground text-sm">{t("routing.defaults.noMatches")}</p>
            ) : null}
          </>
        ) : data.defaults.isSuccess ? (
          <p className="text-muted-foreground text-sm">{t("routing.defaults.empty")}</p>
        ) : null}
        <p className="text-muted-foreground text-xs leading-5">{t("routing.defaults.policy")}</p>
        {priority ? (
          <ProviderPriorityEditor
            open={priorityTarget !== null}
            item={priority}
            onSave={(providerIds) =>
              data.priority.mutateAsync({ model: priority.model, providerIds })
            }
            onClose={() => setPriorityTarget(null)}
            onRestoreFocus={() => {
              if (returnFocus.current?.isConnected && !returnFocus.current.closest("[hidden]"))
                returnFocus.current.focus()
              else
                cardRef.current?.querySelector<HTMLButtonElement>('[aria-pressed="true"]')?.focus()
            }}
          />
        ) : null}
      </CardContent>
    </Card>
  )
}
