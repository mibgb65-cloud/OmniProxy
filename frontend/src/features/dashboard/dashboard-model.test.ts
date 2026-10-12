import { afterEach, describe, expect, it, vi } from "vitest"
import {
  dashboardRange,
  dashboardTokenRatio,
  formatDashboardLatency,
  dashboardIssues,
} from "./dashboard-model"
import { accounts, providers } from "../quota/quota-test-helpers"
import { describeQuotaAccounts, makeQuotaEntry } from "../quota/quota-model"

afterEach(() => vi.unstubAllEnvs())

describe("dashboard local ranges", () => {
  it("uses Singapore local midnight for today and calendar dates for longer periods", () => {
    vi.stubEnv("TZ", "Asia/Singapore")
    const now = new Date("2026-10-09T08:30:00Z")
    const today = dashboardRange(1, now).boundaries!
    expect(today).toHaveLength(25)
    expect(today[0]).toBe(Date.parse("2026-10-08T16:00:00Z") / 1000)
    expect(today.at(-1)).toBe(Date.parse("2026-10-09T16:00:00Z") / 1000)
    expect(dashboardRange(7, now).boundaries).toHaveLength(8)
    expect(dashboardRange(30, now).boundaries).toHaveLength(31)
  })
  it("accounts for 23-hour and 25-hour daylight-saving days", () => {
    vi.stubEnv("TZ", "America/New_York")
    expect(dashboardRange(1, new Date("2026-03-08T12:00:00-04:00")).boundaries).toHaveLength(24)
    expect(dashboardRange(1, new Date("2026-11-01T12:00:00-05:00")).boundaries).toHaveLength(26)
  })
  it("preserves chart scaling for tokens larger than Number's safe integer range", () => {
    expect(dashboardTokenRatio("900719925474099312345", 1801439850948198624690n)).toBe(50)
    expect(dashboardTokenRatio("0", 0n)).toBe(0)
    expect(formatDashboardLatency(null, "zh")).toBe("—")
    expect(formatDashboardLatency(1500, "zh")).toBe("1.5 s")
  })
  it("does not flag deliberately disabled accounts or disabled routes as problems", () => {
    const items = describeQuotaAccounts(accounts, providers)
    const resource = makeQuotaEntry(
      items[3]!,
      {
        pending: false,
        fetching: false,
        error: null,
        balance: {
          available: false,
          entries: [{ currency: "CNY", total: "0", granted: "0", toppedUp: "0" }],
          fetchedAt: "",
        },
      },
      Date.now(),
    )
    const issues = dashboardIssues(accounts, providers, [], [resource])
    expect(issues.map((item) => item.kind)).toEqual(["account"])
    expect(issues[0]?.name).toBe("out@example.invalid")
  })
})
