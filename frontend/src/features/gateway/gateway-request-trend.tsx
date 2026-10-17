import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { Tooltip } from "../../components/ui/tooltip"
import type { DashboardInfo } from "../../services/dashboard"
import { dashboardPointLabel, type DashboardPeriod } from "../dashboard/dashboard-model"
import { gatewayTrend } from "./gateway-overview-model"

export function GatewayRequestTrend({
  data,
  loading,
  period,
}: {
  data?: DashboardInfo
  loading: boolean
  period: DashboardPeriod
}) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const [active, setActive] = useState<number | null>(null)
  const points = data?.points ?? []
  const plot = gatewayTrend(points)
  const hasRequests = points.some((point) => point.summary.requests > 0)
  const number = (value: number) => new Intl.NumberFormat(language).format(value)
  return (
    <Card className="min-w-0 gap-4 py-4">
      <CardHeader className="space-y-2 px-4">
        <h2 className="text-base font-semibold">{t("gateway.trend.title")}</h2>
        <p className="text-muted-foreground text-xs">{t("gateway.trend.hint")}</p>
        <div className="text-muted-foreground flex gap-4 text-xs">
          <span className="inline-flex items-center gap-1.5">
            <span className="h-0.5 w-3 bg-blue-500" aria-hidden="true" />
            {t("activity.success")}
          </span>
          <span className="inline-flex items-center gap-1.5">
            <span className="h-0.5 w-3 bg-rose-500" aria-hidden="true" />
            {t("activity.failedState")}
          </span>
        </div>
      </CardHeader>
      <CardContent className="min-w-0 space-y-3 px-4">
        {hasRequests ? (
          <div className="min-w-0 overflow-x-auto">
            <div style={{ minWidth: Math.max(320, points.length * 24) }}>
              <div className="relative h-52">
                <svg
                  viewBox="0 0 1000 200"
                  preserveAspectRatio="none"
                  aria-hidden="true"
                  className="absolute inset-0 size-full overflow-visible"
                >
                  {[10, 55, 100, 145, 190].map((y) => (
                    <line
                      key={y}
                      x1="0"
                      x2="1000"
                      y1={y}
                      y2={y}
                      className="stroke-border"
                      strokeDasharray="4 5"
                      vectorEffect="non-scaling-stroke"
                    />
                  ))}
                  <path d={plot.area} className="fill-blue-500/10" />
                  <path
                    d={plot.success}
                    fill="none"
                    className="stroke-blue-500"
                    strokeWidth="2"
                    vectorEffect="non-scaling-stroke"
                  />
                  <path
                    d={plot.failed}
                    fill="none"
                    className="stroke-rose-500"
                    strokeWidth="1.5"
                    vectorEffect="non-scaling-stroke"
                  />
                  {active != null && active < points.length ? (
                    <line
                      x1={plot.x(active)}
                      x2={plot.x(active)}
                      y1="10"
                      y2="190"
                      className="stroke-muted-foreground/40"
                      strokeDasharray="3 4"
                      vectorEffect="non-scaling-stroke"
                    />
                  ) : null}
                </svg>
                <div
                  className="text-muted-foreground pointer-events-none absolute inset-y-0 right-0 flex flex-col justify-between py-0.5 text-[10px] tabular-nums"
                  aria-hidden="true"
                >
                  {[4, 3, 2, 1, 0].map((value) => (
                    <span key={value} className="bg-card/80 pl-1">
                      {number(Math.ceil((plot.maximum * value) / 4))}
                    </span>
                  ))}
                </div>
                <div
                  role="group"
                  aria-label={t("gateway.trend.title")}
                  className="absolute inset-0 grid"
                  style={{ gridTemplateColumns: `repeat(${points.length},minmax(0,1fr))` }}
                >
                  {points.map((point, index) => {
                    const title = dashboardPointLabel(point.start, period, language, true)
                    const description = `${title} · ${t("activity.success")} ${number(point.summary.success)} · ${t("activity.failedState")} ${number(point.summary.failed)} · ${t("activity.cancelled")} ${number(point.summary.cancelled)} · ${t("activity.pending")} ${number(point.summary.pending)}`
                    return (
                      <Tooltip
                        key={point.start}
                        content={
                          <div className="space-y-1">
                            <p>{title}</p>
                            {(["success", "failed", "cancelled", "pending"] as const).map((key) => (
                              <p key={key}>
                                {t(key === "failed" ? "activity.failedState" : `activity.${key}`)}：
                                {number(point.summary[key])}
                              </p>
                            ))}
                          </div>
                        }
                      >
                        <div
                          tabIndex={0}
                          aria-label={description}
                          className="focus-visible:ring-ring/60 min-w-0 rounded-sm outline-none focus-visible:ring-2"
                          onMouseEnter={() => setActive(index)}
                          onMouseLeave={() => setActive(null)}
                          onFocus={() => setActive(index)}
                          onBlur={() => setActive(null)}
                        />
                      </Tooltip>
                    )
                  })}
                </div>
              </div>
              <div
                className="text-muted-foreground mt-2 grid text-center text-[10px] tabular-nums"
                style={{ gridTemplateColumns: `repeat(${points.length},minmax(0,1fr))` }}
              >
                {points.map((point, index) => (
                  <span key={point.start}>
                    {index === 0 ||
                    index === points.length - 1 ||
                    index % Math.max(1, Math.ceil(points.length / 6)) === 0
                      ? dashboardPointLabel(point.start, period, language)
                      : ""}
                  </span>
                ))}
              </div>
            </div>
          </div>
        ) : (
          <p className="text-muted-foreground flex h-60 items-center justify-center text-sm">
            {t(
              loading
                ? "gateway.loading"
                : data
                  ? "gateway.trend.empty"
                  : "gateway.overview.unavailable",
            )}
          </p>
        )}
        {hasRequests && plot.peak && plot.peak.summary.success + plot.peak.summary.failed > 0 ? (
          <p className="text-muted-foreground border-t pt-3 text-xs">
            {t("gateway.trend.peak", {
              time: dashboardPointLabel(plot.peak.start, period, language),
              count: number(plot.peak.summary.success + plot.peak.summary.failed),
            })}
          </p>
        ) : null}
      </CardContent>
    </Card>
  )
}
