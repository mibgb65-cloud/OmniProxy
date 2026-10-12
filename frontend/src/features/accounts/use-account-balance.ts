import { queryOptions, useQuery } from "@tanstack/react-query"
import type { AccountItem } from "../../services/omniproxy"
import { readAccountBalance } from "../../services/balance"
import { queueBalanceQuery } from "../../lib/query-queue"

export function accountBalanceOptions(account: AccountItem) {
  return queryOptions({
    // 元数据版本变化可能来自 Key 替换，不把旧凭据对应的余额用于新凭据。
    queryKey: ["account-balance", account.id, account.updatedAt],
    queryFn: ({ signal }) => queueBalanceQuery(() => readAccountBalance(account.id), signal),
    retry: false,
    staleTime: 60_000,
    gcTime: 10 * 60_000,
  })
}

export function useAccountBalance(account: AccountItem) {
  return useQuery(accountBalanceOptions(account))
}
