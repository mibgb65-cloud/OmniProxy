import { useQueries } from "@tanstack/react-query"
import { useState } from "react"
import { accountBalanceOptions } from "../accounts/use-account-balance"
import { subscriptionQuotaOptions } from "../accounts/use-subscription-quota"
import { retainQuota } from "../accounts/quota-retention"
import { googleSubscriptionQuotaOptions } from "../google/use-google-subscription-quota"
import { describeQuotaAccounts, makeQuotaEntry, quotaSummary } from "../quota/quota-model"
import type { AccountItem, ProviderConfiguration } from "../../services/omniproxy"

export function useDashboardResources(
  accounts: AccountItem[],
  providers: ProviderConfiguration[],
  now: number,
) {
  const [checking, setChecking] = useState(false)
  const [result, setResult] = useState<{ total: number; failed: number } | null>(null)
  const items = describeQuotaAccounts(accounts, providers)
  const subscriptions = items.filter((item) => item.kind === "subscription")
  const googleSubscriptions = items.filter((item) => item.kind === "google_subscription")
  const googleQueries = useQueries({
    queries: googleSubscriptions.map((item) => ({
      ...googleSubscriptionQuotaOptions(item.account),
      enabled: false,
    })),
  })
  const balances = items.filter((item) => item.kind === "balance")
  // 只订阅共享缓存，首页挂载不会批量调用官方接口；主动检查复用同一查询与队列。
  const quotas = useQueries({
    queries: subscriptions.map((item) => ({
      ...subscriptionQuotaOptions(item.account),
      enabled: false,
    })),
  })
  const balanceQueries = useQueries({
    queries: balances.map((item) => ({ ...accountBalanceOptions(item.account), enabled: false })),
  })
  const quotaById = new Map(subscriptions.map((item, index) => [item.account.id, quotas[index]]))
  const googleById = new Map(
    googleSubscriptions.map((item, index) => [item.account.id, googleQueries[index]]),
  )
  const balanceById = new Map(
    balances.map((item, index) => [item.account.id, balanceQueries[index]]),
  )
  const entries = items.map((item) => {
    const google = googleById.get(item.account.id)
    const quota = quotaById.get(item.account.id)
    const balance = balanceById.get(item.account.id)
    const query = google ?? quota ?? balance
    // 首页只订阅共享缓存；瞬时失败同样沿用上次成功结果，避免把旧额度计为当前确认值。
    // 判定时刻取失败发生的时刻（errorUpdatedAt），避免在渲染期读时钟。
    const retainedQuota = retainQuota({
      data: quota?.data,
      dataUpdatedAt: quota?.dataUpdatedAt ?? 0,
      isError: quota?.isError ?? false,
      error: quota?.error ?? null,
      now: quota?.errorUpdatedAt ?? 0,
    })
    return makeQuotaEntry(
      item,
      {
        googleQuota: google?.data,
        quota:
          retainedQuota.retained && retainedQuota.data
            ? { ...retainedQuota.data, fromCache: true }
            : retainedQuota.data,
        balance: balance?.data,
        pending: !!query?.isPending && !!query?.isFetching,
        fetching: query?.isFetching ?? false,
        error: retainedQuota.retained ? null : (query?.error ?? null),
        refresh: item.canQuery && query ? () => query.refetch({ cancelRefetch: false }) : undefined,
      },
      now,
    )
  })
  const eligible = entries.filter((item) => item.canQuery && item.refresh)
  const busy = checking || eligible.some((item) => item.fetching)
  async function check() {
    if (busy || eligible.length === 0) return
    setChecking(true)
    setResult(null)
    try {
      const results = await Promise.allSettled(eligible.map((item) => item.refresh!()))
      setResult({
        total: results.length,
        failed: results.filter((item) => item.status === "rejected" || item.value.isError).length,
      })
    } finally {
      setChecking(false)
    }
  }
  const dates = entries.map((item) => Date.parse(item.fetchedAt)).filter(Number.isFinite)
  return {
    entries,
    summary: quotaSummary(entries),
    checked: entries.filter((item) => item.googleQuota || item.quota || item.balance).length,
    lastChecked: dates.length ? new Date(Math.max(...dates)).toISOString() : "",
    busy,
    result,
    check,
    canCheck: eligible.length > 0 && !busy,
  }
}

export type DashboardResources = ReturnType<typeof useDashboardResources>
