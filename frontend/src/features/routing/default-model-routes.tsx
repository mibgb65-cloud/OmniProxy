import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { Input } from "../../components/ui/input"
import { DefaultModelRouteRow } from "./default-model-route-row"
import type { RoutingData } from "./use-routing"

export function DefaultModelRoutes({ data }: { data: RoutingData }) {
  const { t } = useTranslation()
  const [search, setSearch] = useState("")
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
    data.create.isPending ||
    data.update.isPending ||
    data.toggle.isPending ||
    data.remove.isPending ||
    data.defaults.isFetching
  return (
    <Card className="min-w-0 gap-4" aria-label={t("routing.defaults.title")}>
      <CardHeader className="gap-2">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-base font-semibold">{t("routing.defaults.title")}</h2>
          <span className="text-muted-foreground text-xs">
            {t("routing.defaults.count", { count: items.length })}
          </span>
        </div>
        <p className="text-muted-foreground text-sm">{t("routing.defaults.description")}</p>
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
        {items.length ? (
          <>
            <Input
              aria-label={t("routing.defaults.search")}
              placeholder={t("routing.searchPlaceholder")}
              maxLength={128}
              value={search}
              onChange={(event) => setSearch(event.target.value)}
            />
            <ul
              className="divide-border min-w-0 divide-y rounded-lg border"
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
                />
              ))}
            </ul>
            {!visible.length ? (
              <p className="text-muted-foreground text-sm">{t("routing.defaults.noMatches")}</p>
            ) : null}
          </>
        ) : data.defaults.isSuccess ? (
          <p className="text-muted-foreground text-sm">{t("routing.defaults.empty")}</p>
        ) : null}
        <p className="text-muted-foreground text-xs leading-5">{t("routing.defaults.policy")}</p>
      </CardContent>
    </Card>
  )
}
