import { useTranslation } from "react-i18next"
import { Tooltip } from "../../components/ui/tooltip"
import { chartAxisTicks, chartCompact, chartLabelStep, chartPercent } from "../../lib/chart"
import { formatTokens, type Breakdown, type ModelDay } from "../../services/activity"
import { ChartLegend, UsageCard, UsageEmpty } from "./usage-charts"
import { MODEL_COLORS, OTHER_COLOR, usageModelTrend, usageRecentSeries } from "./usage-model"

const COLUMN =
  "group relative flex h-full min-w-0 flex-col justify-end rounded-sm outline-none focus-visible:ring-2 focus-visible:ring-ring/50"

export function UsageModelTrend({ modelDays, days }: { modelDays: ModelDay[]; days: Breakdown[] }) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const points = usageRecentSeries(days, 30)
  const trend = usageModelTrend(modelDays, points)
  const maximum = trend.points.reduce(
    (max, point) => (BigInt(point.total) > max ? BigInt(point.total) : max),
    0n,
  )
  const ticks = chartAxisTicks(maximum)
  const step = chartLabelStep(points.length)
  const compact = (value: bigint) => chartCompact(value, language)
  const colors = new Map(trend.models.map((item, index) => [item.model, MODEL_COLORS[index]!]))
  const legend = trend.models.map((item) => ({
    key: item.model,
    label: item.model || t("activity.noModel"),
    color: colors.get(item.model)!,
    value: compact(BigInt(item.total)),
  }))
  return (
    <UsageCard title={t("activity.modelTrend")} hint={t("activity.modelTrendHint")}>
      {maximum === 0n ? (
        <UsageEmpty>{t("activity.modelTrendEmpty")}</UsageEmpty>
      ) : (
        <div className="space-y-3">
          <ChartLegend
            items={
              trend.hasOther
                ? [
                    ...legend,
                    {
                      key: "__other",
                      label: t("activity.other"),
                      color: OTHER_COLOR,
                    },
                  ]
                : legend
            }
          />
          <div className="flex min-w-0 gap-2">
            <div aria-hidden="true" className="relative h-56 w-10 shrink-0">
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
              <div className="relative h-56">
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
                  aria-label={t("activity.modelTrend")}
                  className="relative grid h-full items-end gap-1"
                  style={{ gridTemplateColumns: `repeat(${points.length}, minmax(0,1fr))` }}
                >
                  {trend.points.map((point) => {
                    const total = BigInt(point.total)
                    return (
                      <Tooltip
                        key={point.key}
                        content={
                          <div className="min-w-48 space-y-1.5">
                            <p className="font-medium">{point.key}</p>
                            {point.segments.map((segment) => (
                              <p
                                key={segment.model}
                                className="flex items-center justify-between gap-6"
                              >
                                <span className="inline-flex min-w-0 items-center gap-1.5">
                                  <span
                                    className={`size-1.5 shrink-0 rounded-full ${colors.get(segment.model)!}`}
                                    aria-hidden="true"
                                  />
                                  <span className="truncate">
                                    {segment.model || t("activity.noModel")}
                                  </span>
                                </span>
                                <span className="shrink-0 tabular-nums">
                                  {formatTokens(segment.value, language)}
                                </span>
                              </p>
                            ))}
                            {BigInt(point.other) > 0n ? (
                              <p className="flex items-center justify-between gap-6">
                                <span className="inline-flex items-center gap-1.5">
                                  <span
                                    className={`size-1.5 rounded-full ${OTHER_COLOR}`}
                                    aria-hidden="true"
                                  />
                                  {t("activity.other")}
                                </span>
                                <span className="tabular-nums">
                                  {formatTokens(point.other, language)}
                                </span>
                              </p>
                            ) : null}
                            <p className="flex items-center justify-between gap-6 border-t pt-1.5 font-medium">
                              <span>{t("activity.totalTokens")}</span>
                              <span className="tabular-nums">
                                {formatTokens(point.total, language)}
                              </span>
                            </p>
                          </div>
                        }
                      >
                        <div
                          tabIndex={0}
                          aria-label={`${point.key} · ${t("activity.totalTokens")} ${formatTokens(point.total, language)}`}
                          className={COLUMN}
                        >
                          <span
                            aria-hidden="true"
                            className="group-hover:bg-foreground/5 group-focus-visible:bg-foreground/5 absolute inset-x-0.5 inset-y-0 rounded-sm transition-colors"
                          />
                          {total > 0n ? (
                            <div
                              className="relative mx-auto flex w-[62%] max-w-9 min-w-[3px] flex-col-reverse overflow-hidden rounded-t-[3px]"
                              style={{
                                height: `${chartPercent(total, ticks[0]!)}%`,
                                minHeight: 2,
                              }}
                            >
                              {point.segments.map((segment) => (
                                <div
                                  key={segment.model}
                                  className={colors.get(segment.model)!}
                                  style={{
                                    height: `${chartPercent(BigInt(segment.value), total)}%`,
                                    minHeight: 1,
                                  }}
                                />
                              ))}
                              {BigInt(point.other) > 0n ? (
                                <div
                                  className={OTHER_COLOR}
                                  style={{
                                    height: `${chartPercent(BigInt(point.other), total)}%`,
                                    minHeight: 1,
                                  }}
                                />
                              ) : null}
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
                style={{ gridTemplateColumns: `repeat(${points.length}, minmax(0,1fr))` }}
              >
                {points.map(({ key }, index) => (
                  <span
                    key={key}
                    className={
                      index === 0
                        ? "text-start whitespace-nowrap"
                        : index === points.length - 1
                          ? "text-end whitespace-nowrap"
                          : "text-center whitespace-nowrap"
                    }
                  >
                    {index % step === 0 || index === points.length - 1 ? key.slice(5) : ""}
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
