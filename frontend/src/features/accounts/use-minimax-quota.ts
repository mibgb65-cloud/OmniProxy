import { queryOptions, useQuery } from "@tanstack/react-query"
import type { AccountItem } from "../../services/omniproxy"
import { readMiniMaxQuota, validMiniMaxOptions } from "../../services/minimax"
import { queueSubscriptionQuery } from "../../lib/query-queue"

export function miniMaxQuotaOptions(account: AccountItem) {
  return queryOptions({
    // updatedAt 同时覆盖凭据轮换和配置变化，旧请求结果不能成为新账号资源。
    queryKey: [
      "minimax-quota",
      account.id,
      account.minimax?.region,
      account.minimax?.billingMode,
      account.minimax?.version,
      account.updatedAt,
    ],
    queryFn: ({ signal }) => queueSubscriptionQuery(() => readMiniMaxQuota(account.id), signal),
    enabled:
      validMiniMaxOptions(account.minimax) &&
      account.minimax?.billingMode === "subscription" &&
      account.authMethod === "api_key",
    retry: false,
    staleTime: 60_000,
    gcTime: 10 * 60_000,
  })
}

export function useMiniMaxQuota(account: AccountItem) {
  return useQuery(miniMaxQuotaOptions(account))
}
