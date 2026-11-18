import { describe, expect, it } from "vitest"
import {
  USAGE_CALENDAR_DAYS,
  USAGE_CALENDAR_WEEKDAYS,
  usageCalendarGrid,
  usageCalendarLevel,
  usageCalendarThresholds,
  usageCalendarTotals,
  usageCalendarValue,
} from "./usage-calendar-model"

function day(key: string, tokens: number, requests = 1) {
  return {
    day: key,
    inputTokens: String(tokens),
    outputTokens: "0",
    totalTokens: String(tokens),
    requests,
  }
}

function key(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`
}

describe("usage calendar grid", () => {
  const now = new Date(2026, 10, 18) // 周三，本地时间

  it("covers the last 365 local days, aligned to whole Monday-first weeks", () => {
    const grid = usageCalendarGrid([], now)
    const cells = grid.columns.flat()
    expect(grid.columns).toHaveLength(53)
    expect(cells).toHaveLength(53 * USAGE_CALENDAR_WEEKDAYS)
    // 起点前补到整周，因此非未来格子 = 补齐天数 + 365，最后一格正好是今天。
    const start = new Date(now)
    start.setDate(start.getDate() - (USAGE_CALENDAR_DAYS - 1))
    const lead = (start.getDay() + 6) % 7
    expect(cells.filter((cell) => !cell.future)).toHaveLength(lead + USAGE_CALENDAR_DAYS)
    const last = cells.filter((cell) => !cell.future).at(-1)!
    expect(last.key).toBe(key(now))
    expect(cells.at(-1)!.future).toBe(true)
    // 每列 7 天，第一列第一行的星期必须是周一。
    const first = new Date(`${cells[0]!.key}T00:00:00`)
    expect((first.getDay() + 6) % 7).toBe(0)
  })

  it("fills missing days with real zeros and keeps recorded days", () => {
    const todayKey = key(now)
    const grid = usageCalendarGrid([day(todayKey, 120, 3)], now)
    const cells = grid.columns.flat()
    const today = cells.find((cell) => cell.key === todayKey)
    expect(today?.day.requests).toBe(3)
    expect(today?.future).toBe(false)
    const filled = cells.find((cell) => cell.key !== todayKey && !cell.future)!
    expect(filled.day).toEqual({
      day: filled.key,
      inputTokens: "0",
      outputTokens: "0",
      totalTokens: "0",
      requests: 0,
    })
  })

  it("labels each month once and keeps labels apart", () => {
    const grid = usageCalendarGrid([], now)
    const columns = grid.months.map((item) => item.column)
    expect(columns.length).toBeGreaterThanOrEqual(10)
    expect(columns.length).toBeLessThanOrEqual(13)
    expect(columns[0]).toBe(0)
    for (let index = 1; index < columns.length; index += 1) {
      expect(columns[index]! - columns[index - 1]!).toBeGreaterThanOrEqual(3)
    }
  })
})

describe("usage calendar levels", () => {
  it("splits non-zero days into quartiles so one spike does not flatten the rest", () => {
    const values = [0n, 1n, 2n, 3n, 4n, 100n]
    const thresholds = usageCalendarThresholds(values)
    expect(thresholds).toHaveLength(3)
    expect(usageCalendarLevel(0n, thresholds)).toBe(0)
    expect(usageCalendarLevel(1n, thresholds)).toBe(1)
    expect(usageCalendarLevel(100n, thresholds)).toBe(4)
    // 阈值本身落在下一档之外：只有严格大于才升档。
    expect(usageCalendarLevel(thresholds[0]!, thresholds)).toBe(1)
    expect(usageCalendarLevel(thresholds[0]! + 1n, thresholds)).toBe(2)
  })

  it("stays neutral when nothing has been recorded", () => {
    const thresholds = usageCalendarThresholds([0n, 0n])
    expect(thresholds).toEqual([])
    expect(usageCalendarLevel(0n, thresholds)).toBe(0)
  })

  it("reads the metric the caller switched to", () => {
    expect(usageCalendarValue(day("2026-11-18", 50, 7), "tokens")).toBe(50n)
    expect(usageCalendarValue(day("2026-11-18", 50, 7), "requests")).toBe(7n)
  })
})

describe("usage calendar totals", () => {
  it("sums only the days the calendar covers", () => {
    expect(usageCalendarTotals([day("2026-11-17", 10, 2), day("2026-11-18", 5, 1)])).toEqual({
      tokens: 15n,
      requests: 3,
    })
  })
})
