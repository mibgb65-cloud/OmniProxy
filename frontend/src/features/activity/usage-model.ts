import { CHART_SERIES_COLORS, chartPercent, chartPercentLabel } from "../../lib/chart"
import type {
  Breakdown,
  CacheUsage,
  HourUsage,
  ModelDay,
  Reliability,
  Summary,
} from "../../services/activity"

/** 模型趋势最多展示六款模型，颜色顺序固定，图例与堆叠段使用同一组 class。 */
export const MODEL_COLORS = CHART_SERIES_COLORS

export const OTHER_COLOR = "bg-muted-foreground/40"

export const USAGE_DAY_RANGES = [7, 14, 30] as const
export type UsageDayRange = (typeof USAGE_DAY_RANGES)[number]
export type UsageMetric = "tokens" | "requests"

export const EMPTY_SUMMARY: Summary = {
  inputTokens: "0",
  outputTokens: "0",
  totalTokens: "0",
  requests: 0,
  completeUsage: 0,
}

export const EMPTY_RELIABILITY: Reliability = {
  requests: 0,
  success: 0,
  failed: 0,
  cancelled: 0,
  successRate: null,
  averageLatencyMs: null,
  averageFirstTokenMs: null,
  firstTokenSamples: 0,
}

export const EMPTY_CACHE: CacheUsage = { cachedInputTokens: "0", knownRequests: 0 }

/** 用量页的日期与桶都按本地日期归属，这里同样用本地年月日生成连续序列。 */
export function usageDateKey(date: Date) {
  const month = String(date.getMonth() + 1).padStart(2, "0")
  const day = String(date.getDate()).padStart(2, "0")
  return `${date.getFullYear()}-${month}-${day}`
}

function emptyBreakdown(key: string): Breakdown {
  return { key, name: "", email: "", workspaceId: "", summary: EMPTY_SUMMARY }
}

/**
 * 近 N 个本地日期的连续序列：没有请求的日期补零，
 * 避免图表把「昨天」和「三周前」画成相邻柱子。
 */
export function usageRecentSeries(days: Breakdown[], range: number, now = new Date()) {
  const byKey = new Map(days.map((day) => [day.key, day]))
  const cursor = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  return Array.from({ length: range }, (_, index) => {
    const date = new Date(cursor)
    date.setDate(cursor.getDate() - (range - 1 - index))
    const key = usageDateKey(date)
    return { key, day: byKey.get(key) ?? emptyBreakdown(key) }
  })
}

/** 用 BigInt 求和，超出 Number 安全范围也不会丢精度。 */
export function usageSum(items: ReadonlyArray<{ summary: Summary }>): Summary {
  let input = 0n
  let output = 0n
  let requests = 0
  let complete = 0
  for (const item of items) {
    input += BigInt(item.summary.inputTokens)
    output += BigInt(item.summary.outputTokens)
    requests += item.summary.requests
    complete += item.summary.completeUsage
  }
  return {
    inputTokens: input.toString(),
    outputTokens: output.toString(),
    totalTokens: (input + output).toString(),
    requests,
    completeUsage: complete,
  }
}

/** 平均每请求 Token；没有请求时返回 null，不伪装成 0。 */
export function usageRequestAverage(total: string, requests: number) {
  return requests > 0 ? (BigInt(total) / BigInt(requests)).toString() : null
}

export function usageShare(part: string | bigint | number, total: string | bigint | number) {
  return chartPercent(BigInt(part), BigInt(total))
}

export function usagePercentLabel(value: number, language: string) {
  return chartPercentLabel(value, language)
}

export function usageCountLabel(value: number, language: string) {
  return new Intl.NumberFormat(language === "zh" ? "zh-CN" : "en-US").format(value)
}

/** 输入／输出比，保留一位小数；输出为零时不构造无穷大。 */
export function usageInputOutputRatio(input: string, output: string) {
  const outputValue = BigInt(output)
  return outputValue > 0n ? Number((BigInt(input) * 10n) / outputValue) / 10 : null
}

export type UsageDelta = { kind: "up" | "down" | "flat"; percent: number } | { kind: "new" }

/** 与前 30 天比较；没有基线时返回 null，基线为零而本期有量时明确标记为新增。 */
export function usageDelta(current: string, previous: string): UsageDelta | null {
  const currentValue = BigInt(current)
  const previousValue = BigInt(previous)
  if (previousValue === 0n) return currentValue === 0n ? null : { kind: "new" }
  if (currentValue === previousValue) return { kind: "flat", percent: 0 }
  const ratio = Number(((currentValue - previousValue) * 10000n) / previousValue) / 100
  return { kind: ratio > 0 ? "up" : "down", percent: Math.abs(ratio) }
}

export type UsageTrendModel = { model: string; total: string }
export type UsageTrendSegment = { model: string; value: string }
export type UsageTrendPoint = {
  key: string
  segments: UsageTrendSegment[]
  other: string
  total: string
}

/**
 * 模型结构趋势：只画用量最高的若干模型，其余按每日总量补成「其他」，
 * 因此每天各段之和始终等于当日合计，不会因截断而少算。
 */
export function usageModelTrend(
  modelDays: ModelDay[],
  points: ReadonlyArray<{ key: string; day: Breakdown }>,
  limit = 6,
) {
  const totals = new Map<string, bigint>()
  const values = new Map<string, Map<string, string>>()
  for (const item of modelDays) {
    totals.set(item.model, (totals.get(item.model) ?? 0n) + BigInt(item.summary.totalTokens))
    const days = values.get(item.day) ?? new Map<string, string>()
    days.set(item.model, item.summary.totalTokens)
    values.set(item.day, days)
  }
  const models = [...totals]
    .sort((a, b) => (a[1] === b[1] ? a[0].localeCompare(b[0]) : a[1] > b[1] ? -1 : 1))
    .slice(0, limit)
    .map(([model, total]) => ({ model, total: total.toString() }))
  let hasOther = false
  const rows = points.map(({ key, day }) => {
    const segments = models
      .map(({ model }) => ({ model, value: values.get(key)?.get(model) ?? "0" }))
      .filter((segment) => BigInt(segment.value) > 0n)
    const total = BigInt(day.summary.totalTokens)
    const shown = segments.reduce((sum, segment) => sum + BigInt(segment.value), 0n)
    const other = total > shown ? total - shown : 0n
    if (other > 0n) hasOther = true
    return { key, segments, other: other.toString(), total: total.toString() }
  })
  return { models, points: rows, hasOther }
}

/** 近 30 天请求高峰时段；没有请求时返回 null。 */
export function usagePeakHour(hours: HourUsage[]) {
  return hours.reduce<HourUsage | null>(
    (peak, hour) => (hour.summary.requests > (peak?.summary.requests ?? 0) ? hour : peak),
    null,
  )
}
