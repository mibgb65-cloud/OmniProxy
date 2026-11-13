import {
  chartAxisMaximum,
  chartAxisTicks,
  chartDuration,
  chartLabelStep,
  chartPercent,
} from "../../lib/chart"
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
  return chartDuration(value, language)
}

/** 首页只看启用提供商下启用的账号；问题列表与体检清单共用同一口径。 */
function enabledProviderIds(providers: ProviderConfiguration[]) {
  return new Set(providers.filter((item) => item.enabled).map((item) => item.id))
}

function liveAccounts(accounts: AccountItem[], providers: ProviderConfiguration[]) {
  const enabled = enabledProviderIds(providers)
  return accounts.filter((item) => item.enabled && enabled.has(item.providerId))
}

function liveResources(resources: QuotaEntry[], providers: ProviderConfiguration[]) {
  const enabled = enabledProviderIds(providers)
  return resources.filter(
    // 不支持查询的类型没有额度语义，既不算问题也不进体检清单的分母。
    (item) =>
      item.kind !== "unsupported" && item.account.enabled && enabled.has(item.account.providerId),
  )
}

export function dashboardIssues(
  accounts: AccountItem[],
  providers: ProviderConfiguration[],
  routes: RouteItem[],
  resources: QuotaEntry[],
): DashboardIssue[] {
  const issues: DashboardIssue[] = liveAccounts(accounts, providers)
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
  for (const item of liveResources(resources, providers).filter(
    (item) => item.status === "exhausted",
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

export type DashboardCheck = {
  key: "dash.checkSubscriptions" | "dash.checkQuota" | "dash.checkRouting"
  healthy: number
  problems: number
}

/**
 * 无待处理问题时前端展示的三行体检清单：把「0 项」拆成可核对的覆盖面，
 * 避免空状态只剩一句结论而无任何信息量。口径与 `dashboardIssues` 完全一致。
 */
export function dashboardChecklist(
  accounts: AccountItem[],
  providers: ProviderConfiguration[],
  routes: RouteItem[],
  resources: QuotaEntry[],
): DashboardCheck[] {
  const subscriptions = liveAccounts(accounts, providers).filter(
    (item) => item.authMethod === "subscription_login",
  )
  const reauth = subscriptions.filter((item) =>
    ["logged_out", "reauth_required"].includes(item.authState ?? ""),
  )
  const monitored = liveResources(resources, providers)
  const exhausted = monitored.filter((item) => item.status === "exhausted")
  const enabledRoutes = routes.filter((item) => item.rule.enabled)
  const invalidRoutes = enabledRoutes.filter((item) => item.errorCode)
  return [
    {
      key: "dash.checkSubscriptions",
      healthy: subscriptions.length - reauth.length,
      problems: reauth.length,
    },
    {
      key: "dash.checkQuota",
      healthy: monitored.length - exhausted.length,
      problems: exhausted.length,
    },
    {
      key: "dash.checkRouting",
      healthy: enabledRoutes.length - invalidRoutes.length,
      problems: invalidRoutes.length,
    },
  ]
}

export function dashboardTokenRatio(value: string, maximum: bigint) {
  return chartPercent(BigInt(value), maximum)
}

// 仪表盘与用量页共用同一套坐标轴／抽稀算法，保留原有导出名避免调用方和测试改动。
export {
  chartAxisMaximum as dashboardAxisMaximum,
  chartAxisTicks as dashboardAxisTicks,
  chartLabelStep as dashboardLabelStep,
}
