import { Link } from "@tanstack/react-router"
import { ArrowUpRight } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import type { DashboardInfo } from "../../services/dashboard"

const colors = [
  "bg-blue-500",
  "bg-teal-600",
  "bg-violet-500",
  "bg-amber-600",
  "bg-rose-500",
  "bg-slate-500",
]

export function GatewayProviderDistribution({
  data,
  loading,
}: {
  data?: DashboardInfo
  loading: boolean
}) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const providers = data?.providerDistribution ?? []
  const total = providers.reduce((sum, item) => sum + item.requests, 0)
  const rows = providers.slice(0, 5).map((item) => ({
    key: `${item.providerId}:${item.providerName}`,
    name: item.providerName || t("activity.legacyProvider"),
    requests: item.requests,
  }))
  if (providers.length > 5)
    rows.push({
      key: "other",
      name: t("gateway.providers.other"),
      requests: providers.slice(5).reduce((sum, item) => sum + item.requests, 0),
    })
  const format = (value: number) => new Intl.NumberFormat(language).format(value)
  return (
    <Card className="min-w-0 gap-4 py-4">
      <CardHeader className="space-y-2 px-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-base font-semibold">{t("gateway.providers.title")}</h2>
          <Link
            to="/routing"
            className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-xs"
          >
            {t("gateway.providers.routes")}
            <ArrowUpRight className="size-3" aria-hidden="true" />
          </Link>
        </div>
        <p className="text-muted-foreground text-xs">{t("gateway.providers.hint")}</p>
      </CardHeader>
      <CardContent className="space-y-4 px-4">
        <div className="bg-muted/30 rounded-lg border p-3">
          <p className="text-muted-foreground text-xs">{t("gateway.providers.total")}</p>
          <p className="mt-1 text-2xl font-semibold tabular-nums">{data ? format(total) : "—"}</p>
        </div>
        {total > 0 ? (
          <>
            <div className="flex h-1.5 overflow-hidden rounded-full" aria-hidden="true">
              {rows.map((item, index) => (
                <span
                  key={item.key}
                  className={colors[index]}
                  style={{ width: `${(item.requests / total) * 100}%` }}
                />
              ))}
            </div>
            <ul className="space-y-3">
              {rows.map((item, index) => (
                <li key={item.key} className="space-y-1.5">
                  <div className="flex min-w-0 items-center gap-2 text-xs">
                    <span
                      className={`size-2 shrink-0 rounded-sm ${colors[index]}`}
                      aria-hidden="true"
                    />
                    <span className="min-w-0 flex-1 truncate font-medium" title={item.name}>
                      {item.name}
                    </span>
                    <span className="tabular-nums">{format(item.requests)}</span>
                    <span className="text-muted-foreground w-11 text-right tabular-nums">
                      {new Intl.NumberFormat(language, {
                        style: "percent",
                        maximumFractionDigits: 1,
                      }).format(item.requests / total)}
                    </span>
                  </div>
                  <div className="bg-muted h-1 overflow-hidden rounded-full" aria-hidden="true">
                    <div
                      className={`${colors[index]} h-full rounded-full`}
                      style={{ width: `${(item.requests / total) * 100}%` }}
                    />
                  </div>
                </li>
              ))}
            </ul>
          </>
        ) : (
          <p className="text-muted-foreground py-3 text-center text-sm">
            {t(
              loading
                ? "gateway.loading"
                : data
                  ? "gateway.providers.empty"
                  : "gateway.overview.unavailable",
            )}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
