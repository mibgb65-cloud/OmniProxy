import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { CHART_SERIES_COLORS, chartPercentLabel } from "../../lib/chart"
import { cn } from "../../lib/utils"
import { formatTokens } from "../../services/activity"
import type { DashboardModel } from "../../services/dashboard"
import { ShareBar } from "../activity/usage-charts"
import { dashboardModelShares } from "./dashboard-model"

const OTHER_COLOR = "bg-muted-foreground/40"

export function DashboardModelShare({
  models,
  loading,
}: {
  models: DashboardModel[]
  loading: boolean
}) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const rows = dashboardModelShares(models)
  const requests = (value: number) =>
    t("dash.modelShareRequests", { value: new Intl.NumberFormat(language).format(value) })
  return (
    <Card className="flex h-full min-w-0 flex-col gap-3 py-4">
      <CardHeader className="gap-1 px-4">
        <h2 className="text-base font-semibold">{t("dash.modelShare")}</h2>
        <p className="text-muted-foreground text-xs leading-5">{t("dash.modelShareHint")}</p>
      </CardHeader>
      <CardContent className="flex min-h-0 flex-1 flex-col px-4">
        {rows.length ? (
          <ul className="flex flex-1 flex-col justify-between gap-3">
            {rows.map((row, index) => {
              const color =
                row.count === undefined
                  ? CHART_SERIES_COLORS[index % CHART_SERIES_COLORS.length]
                  : OTHER_COLOR
              return (
                <li key={row.key} className="min-w-0 space-y-1.5">
                  <div className="flex items-baseline justify-between gap-3">
                    <span className="flex min-w-0 items-center gap-2">
                      <span
                        className={cn("size-2 shrink-0 rounded-full", color)}
                        aria-hidden="true"
                      />
                      <span className="truncate text-sm font-medium">
                        {row.count === undefined
                          ? row.label
                          : t("dash.modelShareOther", { count: row.count })}
                      </span>
                    </span>
                    <span className="text-muted-foreground shrink-0 font-mono text-xs tabular-nums">
                      {chartPercentLabel(row.percent, language)}
                    </span>
                  </div>
                  <ShareBar percent={row.percent} className={color} />
                  <div className="text-muted-foreground flex justify-between gap-3 text-[11px]">
                    <span className="min-w-0 truncate">{requests(row.requests)}</span>
                    <span className="shrink-0 font-mono tabular-nums">
                      {formatTokens(row.tokens, language)}
                    </span>
                  </div>
                </li>
              )
            })}
          </ul>
        ) : (
          <p className="text-muted-foreground flex min-h-24 items-center justify-center px-4 text-center text-sm">
            {t(loading ? "dash.loading" : "dash.modelShareEmpty")}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
