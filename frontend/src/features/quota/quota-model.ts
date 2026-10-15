import type { MiniMaxQuota } from "../../services/minimax"
import type { AccountItem, ProviderConfiguration } from "../../services/omniproxy"
import type { AccountBalanceInfo } from "../../services/balance"
import type { SubscriptionQuota } from "../../services/subscription"
import { accountDisplayName } from "../accounts/account-display"

export const quotaStatuses = [
  "normal",
  "exhausted",
  "error",
  "stale",
  "unauthenticated",
  "unsupported",
  "pending",
  "unknown",
] as const
export type QuotaStatus = (typeof quotaStatuses)[number]
export type QuotaAccount = {
  account: AccountItem
  provider?: ProviderConfiguration
  kind: "subscription" | "balance" | "google" | "step_plan" | "minimax_plan" | "unsupported"
  canQuery: boolean
}
export type QuotaSnapshot = {
  minimax?: MiniMaxQuota
  quota?: SubscriptionQuota
  balance?: AccountBalanceInfo
  pending: boolean
  fetching: boolean
  error: unknown
  refresh?: () => Promise<{ isError: boolean }>
}
export type QuotaEntry = QuotaAccount & QuotaSnapshot & { status: QuotaStatus; fetchedAt: string }

export function describeQuotaAccounts(
  accounts: AccountItem[],
  providers: ProviderConfiguration[],
): QuotaAccount[] {
  const byId = new Map(providers.map((provider) => [provider.id, provider]))
  return accounts.map((account) => {
    const provider = byId.get(account.providerId)
    const kind =
      provider?.kind === "minimax" &&
      account.minimax?.billingMode === "subscription" &&
      ["cn", "global"].includes(account.minimax.region)
        ? "minimax_plan"
        : provider?.kind === "stepfun" && account.stepfun?.billingMode === "step_plan"
          ? "step_plan"
          : provider?.kind === "stepfun" && account.stepfun?.billingMode === "payg"
            ? "balance"
            : provider?.kind === "openai" && account.authMethod === "subscription_login"
              ? "subscription"
              : provider?.kind === "deepseek" &&
                  (!account.authMethod || account.authMethod === "api_key")
                ? "balance"
                : provider?.kind === "google" && account.authMethod === "api_key"
                  ? "google"
                  : "unsupported"
    const canQuery =
      kind !== "unsupported" &&
      kind !== "google" &&
      kind !== "step_plan" &&
      !(
        kind === "subscription" &&
        ["logged_out", "reauth_required"].includes(account.authState ?? "")
      )
    return { account, provider, kind, canQuery }
  })
}

export function quotaWindows(account: AccountItem, data?: SubscriptionQuota) {
  const fiveHour = data?.fiveHourEnabled ?? ["plus", "team"].includes(account.authPlan ?? "")
  return [
    ...(fiveHour
      ? [
          {
            label: "subscription.quota.fiveHour",
            window: data?.fiveHour,
            color: "bg-blue-600",
          } as const,
        ]
      : []),
    { label: "subscription.quota.weekly", window: data?.weekly, color: "bg-emerald-500" } as const,
  ]
}

export function quotaEntryStatus(
  item: QuotaAccount,
  snapshot: QuotaSnapshot,
  now: number,
): QuotaStatus {
  if (item.kind === "minimax_plan") {
    if (snapshot.error) return "error"
    if (!snapshot.minimax) return snapshot.pending ? "pending" : "unknown"
    const buckets = (snapshot.minimax.buckets ?? []).filter(
      (b) => b.model === "general" || b.model.startsWith("MiniMax-M"),
    )
    const windows = buckets.flatMap((b) => [b.interval, b.weekly])
    if (windows.length === 0 || windows.some((w) => w.status === "unknown")) return "unknown"
    if (windows.some((w) => w.resetsAt && Date.parse(w.resetsAt) <= now)) return "stale"
    return windows.some((w) => w.remainingPercent === 0) ? "exhausted" : "normal"
  }
  if (item.kind === "unsupported") return "unsupported"
  if (item.kind === "google" || item.kind === "step_plan") return "unknown"
  if (!item.canQuery) return "unauthenticated"
  if (snapshot.error) return "error"
  if (!snapshot.quota && !snapshot.balance) return snapshot.pending ? "pending" : "unknown"
  if (item.kind === "balance") {
    if (snapshot.balance?.stepfun) return "unknown"
    if (!snapshot.balance?.entries?.length) return "unknown"
    return snapshot.balance.available ? "normal" : "exhausted"
  }
  const windows = quotaWindows(item.account, snapshot.quota).map((item) => item.window)
  if (
    windows.some(
      (window) =>
        !window ||
        !Number.isFinite(window.remainingPercent) ||
        window.remainingPercent < 0 ||
        window.remainingPercent > 100,
    )
  )
    return "unknown"
  // 已经过官方重置目标的快照不能用于判断当前耗尽，也不能直接推算恢复。
  if (windows.some((window) => window?.resetsAt && Date.parse(window.resetsAt) <= now))
    return "stale"
  return windows.some((window) => window?.remainingPercent === 0) ? "exhausted" : "normal"
}

export function makeQuotaEntry(
  item: QuotaAccount,
  snapshot: QuotaSnapshot,
  now: number,
): QuotaEntry {
  return {
    ...item,
    ...snapshot,
    status: quotaEntryStatus(item, snapshot, now),
    fetchedAt:
      snapshot.minimax?.fetchedAt ?? snapshot.quota?.fetchedAt ?? snapshot.balance?.fetchedAt ?? "",
  }
}

export function quotaSummary(entries: QuotaEntry[]) {
  const known = entries.filter((item) => ["normal", "exhausted"].includes(item.status)).length
  return {
    monitored: entries.filter((item) => item.kind !== "unsupported").length,
    exhausted: known > 0 ? entries.filter((item) => item.status === "exhausted").length : null,
    errors: entries.filter((item) => item.status === "error").length,
    uncertain: entries.filter((item) =>
      ["pending", "unknown", "stale", "unauthenticated"].includes(item.status),
    ).length,
    known,
  }
}

export function filterQuotaEntries(
  entries: QuotaEntry[],
  search: string,
  provider: string,
  status: string,
) {
  const term = search.trim().toLocaleLowerCase()
  return entries.filter(
    (item) =>
      (!provider || item.account.providerId === provider) &&
      (!status || item.status === status) &&
      (!term ||
        [
          accountDisplayName(item.account),
          item.account.name,
          item.account.authAccountId,
          item.account.authPlan,
          item.provider?.name,
        ].some((value) => value?.toLocaleLowerCase().includes(term))),
  )
}

export function groupQuotaEntries(entries: QuotaEntry[]) {
  const groups = new Map<
    string,
    { id: string; provider?: ProviderConfiguration; entries: QuotaEntry[] }
  >()
  for (const entry of entries) {
    const id = entry.account.providerId
    const group = groups.get(id) ?? { id, provider: entry.provider, entries: [] }
    group.entries.push(entry)
    groups.set(id, group)
  }
  return [...groups.values()]
}
