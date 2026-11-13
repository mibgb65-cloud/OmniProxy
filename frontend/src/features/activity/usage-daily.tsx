import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import { chartAxisTicks, chartCompact, chartLabelStep, chartPercent } from "../../lib/chart"
import { formatTokens, type Breakdown } from "../../services/activity"
import { UsageCard, UsageEmpty, ChartLegend } from "./usage-charts"
import {
  USAGE_DAY_RANGES,
  usageCountLabel,
  usageRecentSeries,
  usageSum,
  type UsageDayRange,
  type UsageMetric,
} from "./usage-model"

const COLUMN =
  "group relative flex h-full min-w-0 flex-col justify-end rounded-sm outline-none focus-visible:ring-2 focus-visible:ring-ring/50"

export function UsageDailyChart({ days }: { days: Breakdown[] }) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const [range, setRange] = useState<UsageDayRange>(30)
  const [metric, setMetric] = useState<UsageMetric>("tokens")
  const series = usageRecentSeries(days, range)
  const totals = usageSum(series.map((item) => item.day))
  const values = series.map(({ day }) =>
    metric === "tokens" ? BigInt(day.summary.totalTokens) : BigInt(day.summary.requests),
  )
  const maximum = values.reduce((max, value) => (value > max ? value : max), 0n)
  const ticks = chartAxisTicks(maximum)
  const step = chartLabelStep(series.length)
  const compact = (value: bigint) => chartCompact(value, language)
  return (
    <UsageCard
      title={t("activity.daily")}
      hint={t("activity.dailyHint")}
      action={
        <div className="flex flex-wrap items-center gap-2">
          <div
            role="group"
            aria-label={t("activity.metric")}
            className="bg-muted flex rounded-lg p-0.5"
          >
            {(["tokens", "requests"] as const).map((value) => (
              <Button
                key={value}
                variant={metric === value ? "secondary" : "ghost"}
                size="sm"
                aria-pressed={metric === value}
                className={
                  metric === value
                    ? "bg-background text-foreground shadow-xs"
                    : "text-muted-foreground"
                }
                onClick={() => setMetric(value)}
              >
                {t(value === "tokens" ? "activity.metricTokens" : "activity.metricRequests")}
              </Button>
            ))}
          </div>
          <div
            role="group"
            aria-label={t("activity.range")}
            className="bg-muted flex rounded-lg p-0.5"
          >
            {USAGE_DAY_RANGES.map((value) => (
              <Button
                key={value}
                variant={range === value ? "secondary" : "ghost"}
                size="sm"
                aria-pressed={range === value}
                className={
                  range === value
                    ? "bg-background text-foreground shadow-xs"
                    : "text-muted-foreground"
                }
                onClick={() => setRange(value)}
              >
                {t("activity.rangeDays", { count: value })}
              </Button>
            ))}
          </div>
        </div>
      }
    >
      {maximum === 0n ? (
        <UsageEmpty>{t("activity.empty")}</UsageEmpty>
      ) : (
        <div className="space-y-3">
          <ChartLegend
            items={
              metric === "tokens"
                ? [
                    {
                      key: "input",
                      label: t("activity.input"),
                      color: "bg-blue-500",
                      value: compact(BigInt(totals.inputTokens)),
                    },
                    {
                      key: "output",
                      label: t("activity.output"),
                      color: "bg-emerald-500",
                      value: compact(BigInt(totals.outputTokens)),
                    },
                  ]
                : [
                    {
                      key: "requests",
                      label: t("activity.metricRequests"),
                      color: "bg-blue-500",
                      value: usageCountLabel(totals.requests, language),
                    },
                    {
                      key: "complete",
                      label: t("activity.complete"),
                      color: "bg-blue-500/30",
                      value: usageCountLabel(totals.completeUsage, language),
                    },
                  ]
            }
          />
          <div className="flex min-w-0 gap-2">
            <div aria-hidden="true" className="relative h-52 w-10 shrink-0">
              {ticks.map((value, index) => (
                <span
                  key={value.toString()}
                  className="text-muted-foreground absolute right-0 text-[10px] tabular-nums"
                  style={{
                    top: `${(index * 100) / 4}%`,
                    transform:
                      index === 0
                        ? undefined
                        : index === 4
                          ? "translateY(-100%)"
                          : "translateY(-50%)",
                  }}
                >
                  {compact(value)}
                </span>
              ))}
            </div>
            <div className="min-w-0 flex-1">
              <div className="relative h-52">
                <div className="pointer-events-none absolute inset-0" aria-hidden="true">
                  {ticks.slice(1, -1).map((value, index) => (
                    <div
                      key={value.toString()}
                      className="border-border/70 absolute inset-x-0 border-t border-dashed"
                      style={{ top: `${((index + 1) * 100) / 4}%` }}
                    />
                  ))}
                  <div className="border-border absolute inset-x-0 bottom-0 border-t" />
                </div>
                <div
                  role="group"
                  aria-label={t("activity.daily")}
                  className="relative grid h-full items-end gap-1"
                  style={{ gridTemplateColumns: `repeat(${series.length}, minmax(0,1fr))` }}
                >
                  {series.map(({ key, day }) => {
                    const summary = day.summary
                    const input = BigInt(summary.inputTokens)
                    const output = BigInt(summary.outputTokens)
                    const total = input + output
                    const value = metric === "tokens" ? total : BigInt(summary.requests)
                    const title = `${key}`
                    return (
                      <Tooltip
                        key={key}
                        content={
                          <div className="min-w-44 space-y-1.5">
                            <p className="font-medium">{title}</p>
                            <p className="flex items-center justify-between gap-6">
                              <span>{t("activity.input")}</span>
                              <span className="tabular-nums">
                                {formatTokens(summary.inputTokens, language)}
                              </span>
                            </p>
                            <p className="flex items-center justify-between gap-6">
                              <span>{t("activity.output")}</span>
                              <span className="tabular-nums">
                                {formatTokens(summary.outputTokens, language)}
                              </span>
                            </p>
                            <p className="flex items-center justify-between gap-6 border-t pt-1.5 font-medium">
                              <span>{t("activity.totalTokens")}</span>
                              <span className="tabular-nums">
                                {formatTokens(summary.totalTokens, language)}
                              </span>
                            </p>
                            <p className="text-muted-foreground flex items-center justify-between gap-6">
                              <span>{t("activity.metricRequests")}</span>
                              <span className="tabular-nums">
                                {usageCountLabel(summary.requests, language)}
                              </span>
                            </p>
                          </div>
                        }
                      >
                        <div
                          tabIndex={0}
                          aria-label={`${title} · ${t("activity.totalTokens")} ${formatTokens(summary.totalTokens, language)} · ${t("activity.metricRequests")} ${summary.requests}`}
                          className={COLUMN}
                        >
                          <span
                            aria-hidden="true"
                            className="group-hover:bg-foreground/5 group-focus-visible:bg-foreground/5 absolute inset-x-0.5 inset-y-0 rounded-sm transition-colors"
                          />
                          {value > 0n ? (
                            <div
                              className="relative mx-auto flex w-[62%] max-w-9 min-w-[3px] flex-col-reverse overflow-hidden rounded-t-[3px]"
                              style={{
                                height: `${chartPercent(value, ticks[0]!)}%`,
                                minHeight: 2,
                              }}
                            >
                              {metric === "tokens" ? (
                                <>
                                  <div
                                    className="min-h-0 bg-blue-500"
                                    style={{
                                      height: `${chartPercent(input, total)}%`,
                                      minHeight: input > 0n ? 1 : 0,
                                    }}
                                  />
                                  <div
                                    className="min-h-0 bg-emerald-500"
                                    style={{
                                      height: `${chartPercent(output, total)}%`,
                                      minHeight: output > 0n ? 1 : 0,
                                    }}
                                  />
                                </>
                              ) : (
                                <div className="h-full bg-blue-500" />
                              )}
                            </div>
                          ) : null}
                        </div>
                      </Tooltip>
                    )
                  })}
                </div>
              </div>
              <div
                className="text-muted-foreground mt-2 grid gap-1 text-[10px] tabular-nums"
                style={{ gridTemplateColumns: `repeat(${series.length}, minmax(0,1fr))` }}
              >
                {series.map(({ key }, index) => (
                  <span
                    key={key}
                    className={
                      index === 0
                        ? "text-start whitespace-nowrap"
                        : index === series.length - 1
                          ? "text-end whitespace-nowrap"
                          : "text-center whitespace-nowrap"
                    }
                  >
                    {index % step === 0 || index === series.length - 1 ? key.slice(5) : ""}
                  </span>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </UsageCard>
  )
}
