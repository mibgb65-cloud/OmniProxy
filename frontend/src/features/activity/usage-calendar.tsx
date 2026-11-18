import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import { cn } from "../../lib/utils"
import { formatTokens, type DayUsage } from "../../services/activity"
import { UsageCard } from "./usage-charts"
import {
  USAGE_CALENDAR_LEVELS,
  USAGE_CALENDAR_WEEKDAYS,
  usageCalendarGrid,
  usageCalendarLevel,
  usageCalendarThresholds,
  usageCalendarTotals,
  usageCalendarValue,
  type UsageCalendarCell,
} from "./usage-calendar-model"
import { usageCountLabel, type UsageMetric } from "./usage-model"

const columnTemplate = (weeks: number) => `repeat(${weeks}, minmax(0,1fr))`

function useLocaleFormatters() {
  const { i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const locale = language === "zh" ? "zh-CN" : "en-US"
  return {
    language,
    locale,
    weekday: new Intl.DateTimeFormat(locale, { weekday: "short" }),
    month: new Intl.DateTimeFormat(locale, { month: "short" }),
    day: new Intl.DateTimeFormat(locale, { year: "numeric", month: "short", day: "numeric" }),
  }
}

export function UsageCalendar({ days }: { days: DayUsage[] }) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const [metric, setMetric] = useState<UsageMetric>("tokens")
  const { month, weekday } = useLocaleFormatters()
  const grid = usageCalendarGrid(days)
  const cells = grid.columns.flat()
  const visible = cells.filter((cell) => !cell.future)
  const thresholds = usageCalendarThresholds(
    visible.map((cell) => usageCalendarValue(cell.day, metric)),
  )
  const totals = usageCalendarTotals(visible.map((cell) => cell.day))
  const weeks = grid.columns.length
  const summary = t("activity.calendarSummary", {
    tokens: formatTokens(totals.tokens.toString(), language),
    requests: usageCountLabel(totals.requests, language),
  })
  // 2024-01-01 是周一：借它取本地化的「周一／周三／周五」短名，不新增翻译键。
  const weekdays = Array.from({ length: USAGE_CALENDAR_WEEKDAYS }, (_, index) =>
    weekday.format(new Date(2024, 0, 1 + index)),
  )
  return (
    <UsageCard
      title={t("activity.calendar")}
      hint={t("activity.calendarHint")}
      action={
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
      }
    >
      <div className="flex min-w-0 flex-col gap-1.5">
        <div className="flex gap-[3px] text-[10px] leading-4" aria-hidden="true">
          <span className="w-8 shrink-0" />
          <div
            className="grid min-w-0 flex-1 gap-[3px]"
            style={{ gridTemplateColumns: columnTemplate(weeks) }}
          >
            {grid.columns.map((_, index) => {
              const label = grid.months.find((item) => item.column === index)
              return (
                <span key={index} className="text-muted-foreground whitespace-nowrap">
                  {label ? month.format(new Date(2024, label.month, 1)) : ""}
                </span>
              )
            })}
          </div>
        </div>
        <div className="flex min-w-0 gap-[3px]">
          <div
            className="grid w-8 shrink-0 gap-[3px] text-[10px] leading-none"
            aria-hidden="true"
            style={{ gridTemplateRows: `repeat(${USAGE_CALENDAR_WEEKDAYS}, minmax(0,1fr))` }}
          >
            {weekdays.map((label, index) => (
              <span key={label} className="text-muted-foreground flex items-center">
                {index % 2 === 0 && index < 6 ? label : ""}
              </span>
            ))}
          </div>
          {/* 列数固定为 53，用 53:7 的宽高比让每格保持正方形；行高由容器高度均分，与左侧星期标签同高。 */}
          <div
            role="img"
            aria-label={summary}
            className="grid min-w-0 flex-1 gap-[3px]"
            style={{
              aspectRatio: `${weeks} / ${USAGE_CALENDAR_WEEKDAYS}`,
              gridAutoFlow: "column",
              gridTemplateColumns: columnTemplate(weeks),
              gridTemplateRows: `repeat(${USAGE_CALENDAR_WEEKDAYS}, minmax(0,1fr))`,
            }}
          >
            {cells.map((cell) => (
              <CalendarCell key={cell.key} cell={cell} metric={metric} thresholds={thresholds} />
            ))}
          </div>
        </div>
        <div className="text-muted-foreground flex flex-wrap items-center justify-between gap-x-3 gap-y-1 text-[11px]">
          <span>{summary}</span>
          <span className="flex items-center gap-1">
            {t("activity.calendarLess")}
            {USAGE_CALENDAR_LEVELS.map((level) => (
              <span
                key={level}
                className={cn("size-[10px] rounded-[2px]", level)}
                aria-hidden="true"
              />
            ))}
            {t("activity.calendarMore")}
          </span>
        </div>
      </div>
    </UsageCard>
  )
}

function CalendarCell({
  cell,
  metric,
  thresholds,
}: {
  cell: UsageCalendarCell
  metric: UsageMetric
  thresholds: bigint[]
}) {
  const { t } = useTranslation()
  const { language, day } = useLocaleFormatters()
  if (cell.future) {
    return <span aria-hidden="true" className="rounded-[2px] opacity-0" />
  }
  const value = usageCalendarValue(cell.day, metric)
  const label = day.format(new Date(`${cell.key}T00:00:00`))
  return (
    <Tooltip
      content={
        <div className="min-w-44 space-y-1.5">
          <p className="font-medium">{label}</p>
          <p className="flex items-center justify-between gap-6">
            <span>{t("activity.input")}</span>
            <span className="tabular-nums">{formatTokens(cell.day.inputTokens, language)}</span>
          </p>
          <p className="flex items-center justify-between gap-6">
            <span>{t("activity.output")}</span>
            <span className="tabular-nums">{formatTokens(cell.day.outputTokens, language)}</span>
          </p>
          <p className="flex items-center justify-between gap-6 border-t pt-1.5 font-medium">
            <span>{t("activity.totalTokens")}</span>
            <span className="tabular-nums">{formatTokens(cell.day.totalTokens, language)}</span>
          </p>
          <p className="text-muted-foreground flex items-center justify-between gap-6">
            <span>{t("activity.metricRequests")}</span>
            <span className="tabular-nums">{usageCountLabel(cell.day.requests, language)}</span>
          </p>
        </div>
      }
    >
      <span
        aria-hidden="true"
        className={cn(
          "rounded-[2px]",
          USAGE_CALENDAR_LEVELS[usageCalendarLevel(value, thresholds)],
        )}
      />
    </Tooltip>
  )
}
