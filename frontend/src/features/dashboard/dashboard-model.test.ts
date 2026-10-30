import { afterEach, describe, expect, it, vi } from "vitest"
import {
  dashboardAxisMaximum,
  dashboardAxisTicks,
  dashboardChecklist,
  dashboardLabelStep,
  dashboardRange,
  dashboardTokenRatio,
  formatDashboardLatency,
  dashboardIssues,
} from "./dashboard-model"
import { accounts, providers } from "../quota/quota-test-helpers"
import { describeQuotaAccounts, makeQuotaEntry } from "../quota/quota-model"
import type { RouteItem } from "../../services/routing"

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

describe("dashboard checklist", () => {
  it("counts coverage with the same filters as the issue list", () => {
    const now = Date.now()
    const empty = {
      pending: false,
      fetching: false,
      error: null,
      quota: {
        plan: "pro",
        fiveHourEnabled: false,
        fiveHour: null,
        weekly: { remainingPercent: 0, resetsAt: "2099-10-15T00:00:00Z" },
        fetchedAt: "",
      },
    }
    const entries = describeQuotaAccounts(accounts, providers).map((item) =>
      makeQuotaEntry(
        item,
        item.account.id === "sub-empty" ? empty : { pending: false, fetching: false, error: null },
        now,
      ),
    )
    const routes: RouteItem[] = [
      {
        rule: {
          id: "route-broken",
          alias: "coding",
          providerId: "openai",
          accountId: "",
          model: "gpt-test",
          enabled: true,
          createdAt: "",
          updatedAt: "",
        },
        providerName: "OpenAI",
        accountName: "",
        errorCode: "provider_disabled",
      },
      {
        rule: {
          id: "route-off",
          alias: "paused",
          providerId: "openai",
          accountId: "",
          model: "gpt-test",
          enabled: false,
          createdAt: "",
          updatedAt: "",
        },
        providerName: "OpenAI",
        accountName: "",
        errorCode: "provider_disabled",
      },
    ]
    const checklist = dashboardChecklist(accounts, providers, routes, entries)
    expect(checklist).toEqual([
      // 3 个启用的订阅账号，其中 1 个需要重新登录（禁用的账号不计入）。
      { key: "dash.checkSubscriptions", healthy: 2, problems: 1 },
      // 4 个可监控账号（不支持查询与禁用账号不计入），其中 1 个额度耗尽。
      { key: "dash.checkQuota", healthy: 3, problems: 1 },
      // 只统计启用的路由，1 条异常。
      { key: "dash.checkRouting", healthy: 0, problems: 1 },
    ])
    // 清单的问题数与问题列表条数必须一致，避免「清单说正常、列表有问题」。
    expect(checklist.reduce((sum, item) => sum + item.problems, 0)).toBe(
      dashboardIssues(accounts, providers, routes, entries).length,
    )
  })
})

describe("dashboard trend axis", () => {
  it("rounds the vertical axis to readable steps with headroom", () => {
    expect(dashboardAxisMaximum(0n)).toBe(4n)
    expect(dashboardAxisMaximum(1n)).toBe(4n)
    expect(dashboardAxisMaximum(7n)).toBe(8n)
    expect(dashboardAxisMaximum(588n)).toBe(800n)
    expect(dashboardAxisMaximum(1184n)).toBe(1600n)
    expect(dashboardAxisTicks(588n)).toEqual([800n, 600n, 400n, 200n, 0n])
  })
  it("keeps token axes exact beyond Number's safe integer range", () => {
    const axis = dashboardAxisMaximum(900719925474099312345n)
    expect(axis).toBe(1200000000000000000000n)
    expect(dashboardAxisTicks(900719925474099312345n)).toEqual([
      1200000000000000000000n,
      900000000000000000000n,
      600000000000000000000n,
      300000000000000000000n,
      0n,
    ])
  })
  it("keeps at most six readable labels and always marks the last bucket", () => {
    expect(dashboardLabelStep(0)).toBe(1)
    expect(dashboardLabelStep(1)).toBe(1)
    expect(dashboardLabelStep(7)).toBe(2)
    expect(dashboardLabelStep(24)).toBe(5)
    expect(dashboardLabelStep(30)).toBe(6)
    for (const count of [7, 24, 30]) {
      const step = dashboardLabelStep(count)
      const labels = Array.from({ length: count }, (_, index) => index).filter(
        (index) => index % step === 0 || index === count - 1,
      )
      expect(labels.length).toBeLessThanOrEqual(6)
      expect(labels.at(-1)).toBe(count - 1)
    }
  })
})
