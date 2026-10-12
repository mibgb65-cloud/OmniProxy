import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { Tooltip } from "../../components/ui/tooltip"
import { formatTokens } from "../../services/activity"
import type { DashboardInfo, DashboardSummary } from "../../services/dashboard"
import { dashboardPointLabel, dashboardTokenRatio, type DashboardPeriod } from "./dashboard-model"

function series(summary: DashboardSummary, mode: "requests" | "tokens") {
  return mode === "tokens"
    ? ([
        { key: "activity.input", value: summary.inputTokens, color: "bg-blue-500" },
        { key: "activity.output", value: summary.outputTokens, color: "bg-emerald-500" },
      ] as const)
    : ([
        { key: "activity.success", value: String(summary.success), color: "bg-emerald-500" },
        { key: "activity.failedState", value: String(summary.failed), color: "bg-rose-500" },
        { key: "activity.cancelled", value: String(summary.cancelled), color: "bg-zinc-400" },
        { key: "activity.pending", value: String(summary.pending), color: "bg-blue-500" },
      ] as const)
}

export function DashboardTrend({
  data,
  period,
  loading,
}: {
  data?: DashboardInfo
  period: DashboardPeriod
  loading: boolean
}) {
  const { t, i18n } = useTranslation()
  const [mode, setMode] = useState<"requests" | "tokens">("requests")
  const language = i18n.resolvedLanguage ?? "zh"
  const points = data?.points ?? []
  const totals = points.map((point) =>
    series(point.summary, mode).reduce((sum, item) => sum + BigInt(item.value), 0n),
  )
  const maximum = totals.reduce((max, value) => (value > max ? value : max), 1n)
  const legend = data ? series(data.summary, mode) : []
  return (
    <Card className="min-w-0 gap-3 py-4">
      <CardHeader className="space-y-3 px-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-base font-semibold">{t("dash.trend")}</h2>
          <div
            role="group"
            aria-label={t("dash.trendMode")}
            className="bg-muted flex rounded-lg p-0.5"
          >
            {(["requests", "tokens"] as const).map((value) => (
              <Button
                key={value}
                variant={mode === value ? "secondary" : "ghost"}
                size="sm"
                aria-pressed={mode === value}
                className={
                  mode === value
                    ? "bg-background text-foreground shadow-xs"
                    : "text-muted-foreground"
                }
                onClick={() => setMode(value)}
              >
                {t(value === "requests" ? "dash.trendRequests" : "dash.trendTokens")}
              </Button>
            ))}
          </div>
        </div>
        <div className="text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-xs">
          {legend.map((item) => (
            <span key={item.key} className="inline-flex items-center gap-1.5">
              <span className={`size-2 rounded-sm ${item.color}`} aria-hidden="true" />
              {t(item.key)}
            </span>
          ))}
        </div>
      </CardHeader>
      <CardContent className="min-w-0 px-4">
        {data && totals.some((value) => value > 0n) ? (
          <div
            role="group"
            aria-label={t("dash.trend")}
            className="min-w-0 overflow-x-auto pt-2 pb-2"
          >
            <div
              className="relative h-52 px-3"
              style={{ minWidth: Math.max(260, points.length * 26) }}
            >
              <div
                className="pointer-events-none absolute inset-x-3 top-0 bottom-6 flex flex-col justify-between"
                aria-hidden="true"
              >
                {[4, 3, 2, 1, 0].map((value) => (
                  <div key={value} className="border-border/70 relative border-t border-dashed">
                    <span className="bg-card text-muted-foreground absolute -top-2 right-0 pl-1 text-[10px]">
                      {new Intl.NumberFormat(language, {
                        notation: "compact",
                        maximumFractionDigits: 1,
                      }).format((maximum * BigInt(value)) / 4n)}
                    </span>
                  </div>
                ))}
              </div>
              <div
                className="relative grid h-full items-end gap-1"
                style={{ gridTemplateColumns: `repeat(${points.length}, minmax(0,1fr))` }}
              >
                {points.map((point, index) => {
                  const bars = series(point.summary, mode)
                  const title = dashboardPointLabel(point.start, period, language, true)
                  const detail = (
                    <div className="space-y-1">
                      <p>{title}</p>
                      {bars.map((item) => (
                        <p key={item.key}>
                          {t(item.key)}：{formatTokens(item.value, language)}
                        </p>
                      ))}
                    </div>
                  )
                  return (
                    <Tooltip key={point.start} content={detail}>
                      <div
                        tabIndex={0}
                        aria-label={`${title} · ${t("dash.requests")} ${point.summary.requests} · ${t("dash.tokens")} ${point.summary.totalTokens}`}
                        className="focus-visible:ring-ring/50 flex h-full min-w-0 flex-col justify-end rounded-sm outline-none focus-visible:ring-2"
                      >
                        <div className="flex h-full items-end pb-1">
                          <div
                            className="flex w-full flex-col-reverse overflow-hidden rounded-t-sm"
                            style={{
                              height: `${dashboardTokenRatio(totals[index]!.toString(), maximum)}%`,
                            }}
                          >
                            {bars.map((item) => (
                              <div
                                key={item.key}
                                className={item.color}
                                style={{
                                  height: `${dashboardTokenRatio(item.value, totals[index]!)}%`,
                                  minHeight: BigInt(item.value) > 0n ? 1 : 0,
                                }}
                              />
                            ))}
                          </div>
                        </div>
                        <span className="text-muted-foreground h-6 shrink-0 text-center text-[10px] whitespace-nowrap">
                          {index % Math.ceil(points.length / 8) === 0 || index === points.length - 1
                            ? period === 1
                              ? String(new Date(point.start * 1000).getHours()).padStart(2, "0")
                              : dashboardPointLabel(point.start, period, language)
                            : ""}
                        </span>
                      </div>
                    </Tooltip>
                  )
                })}
              </div>
            </div>
          </div>
        ) : (
          <p
            role={loading ? "status" : undefined}
            className="text-muted-foreground flex min-h-52 items-center justify-center px-4 text-center text-sm"
          >
            {t(
              loading
                ? "dash.loading"
                : data
                  ? mode === "tokens" &&
                    data.summary.requests > 0 &&
                    data.summary.completeUsage === 0
                    ? "dash.trendUsageUnknown"
                    : "dash.trendEmpty"
                  : "dash.trendUnavailable",
            )}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
