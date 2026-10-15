import { miniMaxQuotaOptions } from "../accounts/use-minimax-quota"
import { useQueries, useQueryClient } from "@tanstack/react-query"
import { useEffect, useMemo, useRef, useState } from "react"
import { useAccounts } from "../accounts/use-accounts"
import { useProviders } from "../providers/use-providers"
import { accountBalanceOptions } from "../accounts/use-account-balance"
import { subscriptionQuotaOptions } from "../accounts/use-subscription-quota"
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
  const minimaxPlans = items.filter((item) => item.kind === "minimax_plan")
  const minimaxQueries = useQueries({
    queries: minimaxPlans.map((item) => miniMaxQuotaOptions(item.account)),
  })
  const balances = items.filter((item) => item.kind === "balance")
  const subscriptionQueries = useQueries({
    queries: subscriptions.map((item) => subscriptionQuotaOptions(item.account)),
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
      if (item.kind !== "subscription" || !item.canQuery) continue
      const key = subscriptionQuotaOptions(item.account).queryKey
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
  const minimaxById = new Map(
    minimaxPlans.map((item, index) => [item.account.id, minimaxQueries[index]]),
  )
  const entries = items.map((item) => {
    const minimaxQuery = minimaxById.get(item.account.id)
    const quotaQuery = subscriptionById.get(item.account.id)
    const balanceQuery = balanceById.get(item.account.id)
    const query = minimaxQuery ?? quotaQuery ?? balanceQuery
    return makeQuotaEntry(
      item,
      {
        minimax: minimaxQuery?.data,
        quota: quotaQuery?.data,
        balance: balanceQuery?.data,
        pending: query?.isPending ?? false,
        fetching: query?.isFetching ?? false,
        error: query?.error ?? null,
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
