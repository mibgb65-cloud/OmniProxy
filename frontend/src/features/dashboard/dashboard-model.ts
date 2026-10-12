import type { DashboardQuery } from "../../services/dashboard"
import type { AccountItem, ProviderConfiguration } from "../../services/omniproxy"
import type { RouteItem } from "../../services/routing"
import type { QuotaEntry } from "../quota/quota-model"
import { accountDisplayName } from "../accounts/account-display"

export type DashboardPeriod = 1 | 7 | 30
export type DashboardIssue = {
  key: string
  kind: "account" | "route" | "resource"
  name: string
  providerId?: string
  count?: number
}

/** 本地日历边界转换成 UTC 秒，避免把服务端日期或固定 24 小时当成本地一天。 */
export function dashboardRange(period: DashboardPeriod, now = new Date()): DashboardQuery {
  const start = new Date(now)
  start.setHours(0, 0, 0, 0)
  const end = new Date(start)
  end.setDate(end.getDate() + 1)
  start.setDate(start.getDate() - period + 1)
  const boundaries: number[] = []
  if (period === 1) {
    for (let value = start.getTime(); value < end.getTime(); value += 3_600_000)
      boundaries.push(value / 1000)
  } else {
    for (const value = new Date(start); value < end; value.setDate(value.getDate() + 1))
      boundaries.push(value.getTime() / 1000)
  }
  boundaries.push(end.getTime() / 1000)
  return { boundaries }
}

export function dashboardPointLabel(
  start: number,
  period: DashboardPeriod,
  language: string,
  full = false,
) {
  return new Intl.DateTimeFormat(
    language === "zh" ? "zh-CN" : "en-US",
    full
      ? {
          month: "2-digit",
          day: "2-digit",
          hour: "2-digit",
          minute: "2-digit",
          hourCycle: "h23",
          timeZoneName: "shortOffset",
        }
      : period === 1
        ? { hour: "2-digit", minute: "2-digit", hourCycle: "h23" }
        : { month: "2-digit", day: "2-digit" },
  ).format(new Date(start * 1000))
}

export function formatDashboardLatency(value: number | null | undefined, language: string) {
  if (value === null || value === undefined || !Number.isFinite(value)) return "—"
  const locale = language === "zh" ? "zh-CN" : "en-US"
  return value < 1000
    ? `${new Intl.NumberFormat(locale, { maximumFractionDigits: 0 }).format(value)} ms`
    : `${new Intl.NumberFormat(locale, { maximumFractionDigits: 2 }).format(value / 1000)} s`
}

export function dashboardIssues(
  accounts: AccountItem[],
  providers: ProviderConfiguration[],
  routes: RouteItem[],
  resources: QuotaEntry[],
): DashboardIssue[] {
  const enabledProviders = new Set(providers.filter((item) => item.enabled).map((item) => item.id))
  const relevant = accounts.filter((item) => item.enabled && enabledProviders.has(item.providerId))
  const issues: DashboardIssue[] = relevant
    .filter(
      (item) =>
        item.authMethod === "subscription_login" &&
        ["logged_out", "reauth_required"].includes(item.authState ?? ""),
    )
    .map((item) => ({
      key: `account:${item.id}`,
      kind: "account",
      name: accountDisplayName(item),
      providerId: item.providerId,
    }))
  for (const item of resources.filter(
    (item) =>
      item.account.enabled &&
      enabledProviders.has(item.account.providerId) &&
      item.status === "exhausted",
  )) {
    issues.push({
      key: `resource:${item.account.id}`,
      kind: "resource",
      name: accountDisplayName(item.account),
      providerId: item.account.providerId,
    })
  }
  const invalid = routes.filter((item) => item.rule.enabled && item.errorCode)
  if (invalid.length) issues.push({ key: "route", kind: "route", name: "", count: invalid.length })
  return issues
}

export function dashboardTokenRatio(value: string, maximum: bigint) {
  return maximum > 0n ? Number((BigInt(value) * 10000n) / maximum) / 100 : 0
}
