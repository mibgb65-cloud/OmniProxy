import type { DayUsage } from "../../services/activity"
import { usageDateKey } from "./usage-model"

/** 年历覆盖 365 个本地日期；补到整周后最多 53 列。 */
export const USAGE_CALENDAR_DAYS = 365
export const USAGE_CALENDAR_WEEKDAYS = 7
/** 月份标签之间至少间隔的列数：中文月份比一格宽，挨着放会互相覆盖。 */
const MONTH_LABEL_MIN_COLUMNS = 3

/** 五档色阶：0 为中性底色，四档绿随用量递增；class 必须写成字面量，Tailwind 才能扫描到。 */
export const USAGE_CALENDAR_LEVELS = [
  "bg-muted",
  "bg-emerald-500/20",
  "bg-emerald-500/40",
  "bg-emerald-500/65",
  "bg-emerald-500",
] as const

export type UsageCalendarCell = { key: string; day: DayUsage; future: boolean }
export type UsageCalendarMonth = { column: number; month: number }
export type UsageCalendarGrid = { columns: UsageCalendarCell[][]; months: UsageCalendarMonth[] }

export function emptyUsageDay(day: string): DayUsage {
  return { day, inputTokens: "0", outputTokens: "0", totalTokens: "0", requests: 0 }
}

/** 周一为 0 的星期序号：年历按周一分行，与「周一／周三／周五」标签一致。 */
export function usageWeekdayIndex(date: Date) {
  return (date.getDay() + 6) % 7
}

/**
 * 把按日用量铺成 GitHub 风格的年历：以今天为最后一格，向左补到整周，向右补足最后一周。
 * 今天之后的格子标记为 future（只占位，不显示也不参与色阶），日期没有记录时补零——
 * 「当天没有请求」是真实的 0，不是未知。
 */
export function usageCalendarGrid(days: DayUsage[], now = new Date()): UsageCalendarGrid {
  const byKey = new Map(days.map((item) => [item.day, item]))
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const start = new Date(today)
  start.setDate(today.getDate() - (USAGE_CALENDAR_DAYS - 1))
  const lead = usageWeekdayIndex(start)
  const total = lead + USAGE_CALENDAR_DAYS
  const columnCount = Math.ceil(total / USAGE_CALENDAR_WEEKDAYS)
  const columns: UsageCalendarCell[][] = Array.from({ length: columnCount }, () => [])
  for (let index = 0; index < columnCount * USAGE_CALENDAR_WEEKDAYS; index += 1) {
    const date = new Date(start)
    date.setDate(start.getDate() + (index - lead))
    const key = usageDateKey(date)
    columns[Math.floor(index / USAGE_CALENDAR_WEEKDAYS)]!.push({
      key,
      day: byKey.get(key) ?? emptyUsageDay(key),
      future: date > today,
    })
  }
  const months: UsageCalendarMonth[] = []
  let lastMonth = ""
  columns.forEach((column, index) => {
    const first = column.find((cell) => !cell.future)
    if (!first) return
    const month = first.key.slice(0, 7)
    if (month === lastMonth) return
    lastMonth = month
    const shown = months[months.length - 1]
    if (shown && index - shown.column < MONTH_LABEL_MIN_COLUMNS) return
    months.push({ column: index, month: Number(first.key.slice(5, 7)) - 1 })
  })
  return { columns, months }
}

export function usageCalendarValue(day: DayUsage, metric: "tokens" | "requests") {
  return metric === "tokens" ? BigInt(day.totalTokens) : BigInt(day.requests)
}

/**
 * 色阶阈值取非零日子的四分位：用量分布长尾很重，按最大值线性分档会把绝大多数日子压成同一档。
 */
export function usageCalendarThresholds(values: readonly bigint[]) {
  const sorted = values.filter((value) => value > 0n).sort((a, b) => (a < b ? -1 : a > b ? 1 : 0))
  if (!sorted.length) return [] as bigint[]
  const pick = (ratio: number) =>
    sorted[Math.min(sorted.length - 1, Math.floor(sorted.length * ratio))]!
  return [pick(0.25), pick(0.5), pick(0.75)]
}

export function usageCalendarLevel(value: bigint, thresholds: readonly bigint[]) {
  if (value <= 0n) return 0
  let level = 1
  for (const threshold of thresholds) if (value > threshold) level += 1
  return level
}

/** 年历口径的合计；只统计被年历覆盖的日期，与卡片标题的「过去一年」一致。 */
export function usageCalendarTotals(days: readonly DayUsage[]) {
  let tokens = 0n
  let requests = 0
  for (const day of days) {
    tokens += BigInt(day.totalTokens)
    requests += day.requests
  }
  return { tokens, requests }
}
