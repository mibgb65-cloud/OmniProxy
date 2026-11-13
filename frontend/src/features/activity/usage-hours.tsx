import { useTranslation } from "react-i18next"
import { Tooltip } from "../../components/ui/tooltip"
import { chartAxisTicks, chartCompact, chartPercent } from "../../lib/chart"
import { formatTokens, type HourUsage } from "../../services/activity"
import { UsageCard, UsageEmpty } from "./usage-charts"
import { EMPTY_SUMMARY, usageCountLabel, usagePeakHour } from "./usage-model"

const COLUMN =
  "group relative flex h-full min-w-0 flex-col justify-end rounded-sm outline-none focus-visible:ring-2 focus-visible:ring-ring/50"

export function UsageHours({ hours }: { hours: HourUsage[] }) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const byHour = new Map(hours.map((item) => [item.hour, item]))
  const slots = Array.from(
    { length: 24 },
    (_, hour) => byHour.get(hour) ?? { hour, summary: EMPTY_SUMMARY },
  )
  const maximum = slots.reduce(
    (max, item) => (BigInt(item.summary.requests) > max ? BigInt(item.summary.requests) : max),
    0n,
  )
  const ticks = chartAxisTicks(maximum)
  const peak = usagePeakHour(slots)
  const compact = (value: bigint) => chartCompact(value, language)
  return (
    <UsageCard
      title={t("activity.hours")}
      hint={t("activity.hoursHint")}
      action={
        peak ? (
          <span className="text-muted-foreground text-[11px]">
            {t("activity.peakHour", {
              hour: String(peak.hour).padStart(2, "0"),
              count: usageCountLabel(peak.summary.requests, language),
            })}
          </span>
        ) : null
      }
    >
      {maximum === 0n ? (
        <UsageEmpty>{t("activity.empty")}</UsageEmpty>
      ) : (
        <div className="flex min-w-0 gap-2">
          <div aria-hidden="true" className="relative h-44 w-10 shrink-0">
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
            <div className="relative h-44">
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
                aria-label={t("activity.hours")}
                className="relative grid h-full items-end gap-0.5"
                style={{ gridTemplateColumns: "repeat(24, minmax(0,1fr))" }}
              >
                {slots.map((item) => {
                  const requests = BigInt(item.summary.requests)
                  const isPeak = peak?.hour === item.hour && requests > 0n
                  const label = t("activity.hourLabel", {
                    hour: String(item.hour).padStart(2, "0"),
                  })
                  return (
                    <Tooltip
                      key={item.hour}
                      content={
                        <div className="min-w-44 space-y-1.5">
                          <p className="font-medium">{label}</p>
                          <p className="flex items-center justify-between gap-6">
                            <span>{t("activity.metricRequests")}</span>
                            <span className="tabular-nums">
                              {usageCountLabel(item.summary.requests, language)}
                            </span>
                          </p>
                          <p className="flex items-center justify-between gap-6">
                            <span>{t("activity.totalTokens")}</span>
                            <span className="tabular-nums">
                              {formatTokens(item.summary.totalTokens, language)}
                            </span>
                          </p>
                        </div>
                      }
                    >
                      <div
                        tabIndex={0}
                        aria-label={`${label} · ${t("activity.metricRequests")} ${item.summary.requests} · ${formatTokens(item.summary.totalTokens, language)}`}
                        className={COLUMN}
                      >
                        <span
                          aria-hidden="true"
                          className="group-hover:bg-foreground/5 group-focus-visible:bg-foreground/5 absolute inset-x-0 inset-y-0 rounded-sm transition-colors"
                        />
                        {requests > 0n ? (
                          <div
                            className={`relative mx-auto w-[66%] max-w-6 min-w-[3px] rounded-t-[2px] ${isPeak ? "bg-amber-500" : "bg-blue-500"}`}
                            style={{
                              height: `${chartPercent(requests, ticks[0]!)}%`,
                              minHeight: 2,
                            }}
                          />
                        ) : null}
                      </div>
                    </Tooltip>
                  )
                })}
              </div>
            </div>
            <div
              className="text-muted-foreground mt-2 grid gap-0.5 text-[10px] tabular-nums"
              style={{ gridTemplateColumns: "repeat(24, minmax(0,1fr))" }}
            >
              {slots.map((item) => (
                <span key={item.hour} className="text-center whitespace-nowrap">
                  {item.hour % 3 === 0 ? String(item.hour).padStart(2, "0") : ""}
                </span>
              ))}
            </div>
          </div>
        </div>
      )}
    </UsageCard>
  )
}
