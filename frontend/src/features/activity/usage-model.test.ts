import { describe, expect, it } from "vitest"
import type { Breakdown, HourUsage, ModelDay } from "../../services/activity"
import {
  usageDateKey,
  usageDelta,
  usageInputOutputRatio,
  usageModelTrend,
  usagePeakHour,
  usageRecentSeries,
  usageRequestAverage,
  usageShare,
  usageSum,
} from "./usage-model"

function day(
  key: string,
  input: number | string,
  output: number | string,
  requests = 1,
): Breakdown {
  const inputValue = BigInt(input)
  const outputValue = BigInt(output)
  return {
    key,
    name: "",
    email: "",
    workspaceId: "",
    summary: {
      inputTokens: inputValue.toString(),
      outputTokens: outputValue.toString(),
      totalTokens: (inputValue + outputValue).toString(),
      requests,
      completeUsage: requests,
    },
  }
}

function modelDay(dayKey: string, model: string, total: number): ModelDay {
  return {
    day: dayKey,
    model,
    summary: {
      inputTokens: String(total),
      outputTokens: "0",
      totalTokens: String(total),
      requests: 1,
      completeUsage: 1,
    },
  }
}

describe("usage local day series", () => {
  it("builds consecutive local dates across a month boundary", () => {
    const series = usageRecentSeries([], 3, new Date(2026, 2, 2, 12, 0, 0))
    expect(series.map((item) => item.key)).toEqual(["2026-02-28", "2026-03-01", "2026-03-02"])
    expect(usageDateKey(new Date(2026, 0, 9))).toBe("2026-01-09")
  })

  it("keeps recorded days and zero-fills missing ones", () => {
    const series = usageRecentSeries([day("2026-03-01", 10, 5)], 3, new Date(2026, 2, 2))
    expect(series.map((item) => item.day.summary.totalTokens)).toEqual(["0", "15", "0"])
  })
})

describe("usage arithmetic", () => {
  it("sums beyond Number's safe integer range", () => {
    const summary = usageSum([day("a", "9007199254740993", 0), day("b", 0, 7)])
    expect(summary.totalTokens).toBe("9007199254741000")
    expect(summary.requests).toBe(2)
  })

  it("keeps average and share without dividing by zero", () => {
    expect(usageRequestAverage("100", 3)).toBe("33")
    expect(usageRequestAverage("100", 0)).toBeNull()
    expect(usageShare("1", "3")).toBe(33.33)
    expect(usageShare("1", "0")).toBe(0)
    expect(usageInputOutputRatio("1000", "250")).toBe(4)
    expect(usageInputOutputRatio("1000", "0")).toBeNull()
  })

  it("compares the current window with the previous one", () => {
    expect(usageDelta("150", "100")).toEqual({ kind: "up", percent: 50 })
    expect(usageDelta("50", "100")).toEqual({ kind: "down", percent: 50 })
    expect(usageDelta("100", "100")).toEqual({ kind: "flat", percent: 0 })
    expect(usageDelta("100", "0")).toEqual({ kind: "new" })
    expect(usageDelta("0", "0")).toBeNull()
  })
})

describe("usage model trend", () => {
  it("keeps the top models and folds the rest into other per day", () => {
    const points = usageRecentSeries(
      [day("2026-03-01", 280, 0), day("2026-03-02", 30, 0)],
      2,
      new Date(2026, 2, 2),
    )
    const modelDays = [1, 2, 3, 4, 5, 6, 7].flatMap((value) => [
      modelDay("2026-03-01", `m${value}`, value * 10),
      modelDay("2026-03-02", `m${value}`, value === 7 ? 30 : 0),
    ])
    const trend = usageModelTrend(modelDays, points)
    expect(trend.models.map((item) => item.model)).toEqual(["m7", "m6", "m5", "m4", "m3", "m2"])
    expect(trend.hasOther).toBe(true)
    expect(trend.points[0]?.other).toBe("10")
    expect(trend.points[0]?.total).toBe("280")
    expect(trend.points[1]?.segments).toEqual([{ model: "m7", value: "30" }])
    expect(trend.points[1]?.other).toBe("0")
  })

  it("does not invent other when every model is visible", () => {
    const points = usageRecentSeries([day("2026-03-02", 30, 0)], 1, new Date(2026, 2, 2))
    const trend = usageModelTrend([modelDay("2026-03-02", "m1", 30)], points)
    expect(trend.hasOther).toBe(false)
    expect(trend.points[0]?.other).toBe("0")
  })
})

describe("usage peak hour", () => {
  it("returns the busiest local hour or null when empty", () => {
    const hours: HourUsage[] = Array.from({ length: 24 }, (_, hour) => ({
      hour,
      summary: {
        inputTokens: "0",
        outputTokens: "0",
        totalTokens: "0",
        requests: hour === 9 ? 4 : hour === 14 ? 2 : 0,
        completeUsage: 0,
      },
    }))
    expect(usagePeakHour(hours)?.hour).toBe(9)
    expect(usagePeakHour([])).toBeNull()
    expect(
      usagePeakHour(hours.map((item) => ({ ...item, summary: { ...item.summary, requests: 0 } }))),
    ).toBeNull()
  })
})
