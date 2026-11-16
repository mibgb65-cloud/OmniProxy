import { miniMaxQuotaOptions } from "../accounts/use-minimax-quota"
import { googleSubscriptionQuotaOptions } from "../google/use-google-subscription-quota"
import { duMatePointsOptions } from "../dumate/use-dumate"
import { useQueries, useQueryClient } from "@tanstack/react-query"
import { useEffect, useMemo, useRef, useState } from "react"
import { useAccounts } from "../accounts/use-accounts"
import { useProviders } from "../providers/use-providers"
import { accountBalanceOptions } from "../accounts/use-account-balance"
import { subscriptionQuotaOptions } from "../accounts/use-subscription-quota"
import { workbuddyQuotaOptions } from "../accounts/use-workbuddy-quota"
import { retainQuota } from "../accounts/quota-retention"
import {
  describeQuotaAccounts,
  filterQuotaEntries,
  groupQuotaEntries,
  makeQuotaEntry,
  quotaSummary,
} from "./quota-model"

export function useQuotaOverview() {
  const client = useQueryClient()
  const accounts = useAccounts()
  const providers = useProviders()
  const [search, setSearch] = useState("")
  const [provider, setProvider] = useState("")
  const [status, setStatus] = useState("")
  const [now, setNow] = useState(() => Date.now())
  const [batchBusy, setBatchBusy] = useState(false)
  const [batchResult, setBatchResult] = useState<{ total: number; failed: number } | null>(null)
  const previous = useRef(new Map<string, string>())
  const items = useMemo(
    () => describeQuotaAccounts(accounts.data ?? [], providers.data ?? []),
    [accounts.data, providers.data],
  )
  const subscriptions = items.filter((item) => item.kind === "subscription")
  const googleSubscriptions = items.filter((item) => item.kind === "google_subscription")
  const googleQueries = useQueries({
    queries: googleSubscriptions.map((item) => googleSubscriptionQuotaOptions(item.account)),
  })
  const minimaxPlans = items.filter((item) => item.kind === "minimax_plan")
  const minimaxQueries = useQueries({
    queries: minimaxPlans.map((item) => miniMaxQuotaOptions(item.account)),
  })
  const balances = items.filter((item) => item.kind === "balance")
  // 未登录或需重新导入的 DuMate 账号不发起控制台查询；其余与账号页共用积分查询键。
  const dumateAccounts = items.filter((item) => item.kind === "dumate_points")
  const dumateQueries = useQueries({
    queries: dumateAccounts.map((item) => ({
      ...duMatePointsOptions(item.account.id),
      enabled: item.canQuery,
    })),
  })
  const subscriptionQueries = useQueries({
    queries: subscriptions.map((item) => subscriptionQuotaOptions(item.account)),
  })
  const workbuddyAccounts = items.filter((item) => item.kind === "workbuddy")
  const workbuddyQueries = useQueries({
    queries: workbuddyAccounts.map((item) => workbuddyQuotaOptions(item.account)),
  })
  const balanceQueries = useQueries({
    queries: balances.map((item) => accountBalanceOptions(item.account)),
  })

  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 30_000)
    return () => clearInterval(timer)
  }, [])
  useEffect(() => {
    const versions = new Map<string, string>()
    for (const item of items) {
      if (
        !["subscription", "google_subscription", "workbuddy"].includes(item.kind) ||
        !item.canQuery
      )
        continue
      const key =
        item.kind === "google_subscription"
          ? googleSubscriptionQuotaOptions(item.account).queryKey
          : item.kind === "workbuddy"
            ? workbuddyQuotaOptions(item.account).queryKey
            : subscriptionQuotaOptions(item.account).queryKey
      const scope = JSON.stringify(key)
      const oldVersion = previous.current.get(scope)
      versions.set(scope, item.account.updatedAt)
      if (oldVersion !== undefined && oldVersion !== item.account.updatedAt) {
        void client.refetchQueries(
          { queryKey: key, exact: true, type: "active" },
          { cancelRefetch: false },
        )
      }
    }
    previous.current = versions
  }, [client, items])

  const subscriptionById = new Map(
    subscriptions.map((item, index) => [item.account.id, subscriptionQueries[index]]),
  )
  const balanceById = new Map(
    balances.map((item, index) => [item.account.id, balanceQueries[index]]),
  )
  const googleById = new Map(
    googleSubscriptions.map((item, index) => [item.account.id, googleQueries[index]]),
  )
  const minimaxById = new Map(
    minimaxPlans.map((item, index) => [item.account.id, minimaxQueries[index]]),
  )
  const dumateById = new Map(
    dumateAccounts.map((item, index) => [item.account.id, dumateQueries[index]]),
  )
  const workbuddyById = new Map(
    workbuddyAccounts.map((item, index) => [item.account.id, workbuddyQueries[index]]),
  )
  const entries = items.map((item) => {
    const googleQuery = googleById.get(item.account.id)
    const minimaxQuery = minimaxById.get(item.account.id)
    const quotaQuery = subscriptionById.get(item.account.id)
    const balanceQuery = balanceById.get(item.account.id)
    const dumateQuery = dumateById.get(item.account.id)
    const workbuddyQuery = workbuddyById.get(item.account.id)
    const query =
      googleQuery ?? minimaxQuery ?? quotaQuery ?? balanceQuery ?? dumateQuery ?? workbuddyQuery
    // 瞬时失败沿用上次成功结果：错误置空，值标为本地留存，卡片因此显示“已保留上次获取的信息”。
    // 判定时刻取失败发生的时刻（errorUpdatedAt），避免在渲染期读时钟。
    const google = retainQuota({
      data: googleQuery?.data,
      dataUpdatedAt: googleQuery?.dataUpdatedAt ?? 0,
      isError: googleQuery?.isError ?? false,
      error: googleQuery?.error ?? null,
      now: googleQuery?.errorUpdatedAt ?? 0,
    })
    const subscription = retainQuota({
      data: quotaQuery?.data,
      dataUpdatedAt: quotaQuery?.dataUpdatedAt ?? 0,
      isError: quotaQuery?.isError ?? false,
      error: quotaQuery?.error ?? null,
      now: quotaQuery?.errorUpdatedAt ?? 0,
    })
    const workbuddy = retainQuota({
      data: workbuddyQuery?.data,
      dataUpdatedAt: workbuddyQuery?.dataUpdatedAt ?? 0,
      isError: workbuddyQuery?.isError ?? false,
      error: workbuddyQuery?.error ?? null,
      now: workbuddyQuery?.errorUpdatedAt ?? 0,
    })
    const retained = google.retained || subscription.retained || workbuddy.retained
    return makeQuotaEntry(
      item,
      {
        googleQuota:
          google.retained && google.data ? { ...google.data, fromCache: true } : google.data,
        minimax: minimaxQuery?.data,
        quota:
          subscription.retained && subscription.data
            ? { ...subscription.data, fromCache: true }
            : subscription.data,
        balance: balanceQuery?.data,
        dumate: dumateQuery?.data,
        workbuddy:
          workbuddy.retained && workbuddy.data
            ? { ...workbuddy.data, fromCache: true }
            : workbuddy.data,
        // 停用查询（例如未登录的 DuMate）不算「正在读取」，避免空白卡片一直显示占位。
        pending: item.canQuery && (query?.isPending ?? false),
        fetching: query?.isFetching ?? false,
        error: retained ? null : (query?.error ?? null),
        refresh: item.canQuery && query ? () => query.refetch({ cancelRefetch: false }) : undefined,
      },
      now,
    )
  })
  const visible = filterQuotaEntries(entries, search, provider, status)
  const refreshable = visible.filter((item) => item.canQuery && item.refresh)
  const ready = accounts.data !== undefined && providers.data !== undefined
  const refreshing = batchBusy || refreshable.some((item) => item.fetching)

  async function refreshVisible() {
    if (refreshing || !ready || refreshable.length === 0) return
    setBatchBusy(true)
    setBatchResult(null)
    try {
      const results = await Promise.allSettled(refreshable.map((item) => item.refresh!()))
      const failed = results.filter(
        (result) => result.status === "rejected" || result.value.isError,
      ).length
      setBatchResult({ total: results.length, failed })
    } finally {
      setBatchBusy(false)
    }
  }

  async function refreshMetadata() {
    await Promise.all([
      accounts.refetch({ cancelRefetch: false }),
      providers.refetch({ cancelRefetch: false }),
    ])
  }

  return {
    entries,
    groups: groupQuotaEntries(visible),
    summary: quotaSummary(entries),
    ready,
    now,
    search,
    setSearch,
    provider,
    setProvider,
    status,
    setStatus,
    providers: providers.data ?? [],
    visibleCount: visible.length,
    metadataLoading: !ready && (accounts.isPending || providers.isPending),
    metadataError: accounts.isError || providers.isError,
    metadataRefreshing: accounts.isFetching || providers.isFetching,
    refreshMetadata,
    refreshVisible,
    refreshing,
    batchResult,
    canRefresh: ready && refreshable.length > 0 && !refreshing,
    progress: {
      total: entries.filter((item) => item.canQuery).length,
      finished: entries.filter((item) => item.canQuery && !item.pending && !item.fetching).length,
      fetching: entries.some((item) => item.fetching),
    },
  }
}

export type QuotaOverview = ReturnType<typeof useQuotaOverview>
