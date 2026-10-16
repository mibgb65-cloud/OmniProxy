import { describe, expect, it } from "vitest"
import { googleQuotaState, googleQuotaPercent } from "./google-quota-model"
import { googleQuotaFixture } from "./google-quota-test-helpers"

describe("Google official quota state", () => {
  const now = Date.parse("2026-10-16T00:00:00Z")
  it("keeps missing, zero and tiny positive fractions distinct", () => {
    for (const [percent, expected] of [
      [null, "unknown"],
      [0, "exhausted"],
      [0.00001, "normal"],
      [100, "normal"],
    ] as const) {
      const quota = {
        ...googleQuotaFixture,
        groups: [
          {
            name: "Gemini",
            buckets: [{ id: "window", name: "", window: "weekly", remainingPercent: percent }],
          },
        ],
      }
      expect(googleQuotaState(quota, now)).toBe(expected)
    }
    expect(googleQuotaState({ groups: [], fetchedAt: "" }, now)).toBe("unknown")
  })
  it("does not turn expired windows into full quota", () => {
    const quota = {
      ...googleQuotaFixture,
      groups: [
        {
          name: "Gemini",
          buckets: [
            {
              id: "window",
              name: "",
              window: "weekly",
              remainingPercent: 0,
              resetsAt: "2020-01-01T00:00:00Z",
            },
          ],
        },
      ],
    }
    expect(googleQuotaState(quota, now)).toBe("stale")
    expect(quota.groups[0].buckets[0].remainingPercent).toBe(0)
  })
  it("rejects invalid values and avoids misleading rounded percentages", () => {
    expect(googleQuotaPercent(null, "zh")).toBe("—")
    expect(googleQuotaPercent(0, "zh")).toBe("0%")
    expect(googleQuotaPercent(0.0001, "zh")).toBe("<0.1%")
    expect(googleQuotaPercent(99.999, "en")).toBe("99.9%")
    expect(googleQuotaPercent(100, "en")).toBe("100%")
    for (const percent of [-1, 101, NaN, Infinity])
      expect(googleQuotaPercent(percent, "zh")).toBe("—")
  })
})
