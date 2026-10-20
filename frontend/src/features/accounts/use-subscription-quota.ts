import { queryOptions, useQuery, useQueryClient } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import { useEffect, useRef } from "react"
import type { AccountItem } from "../../services/omniproxy"
import {
  readSubscriptionQuota,
  readCachedSubscriptionQuota,
  type SubscriptionQuota,
} from "../../services/subscription"
import { queueSubscriptionQuery } from "../../lib/query-queue"

export function subscriptionQuotaKey(account: AccountItem) {
  return [
    "subscription-quota",
    account.id,
    account.authAccountId || "",
    account.authEmail?.trim().toLowerCase() || "",
    account.authState || "unknown",
  ] as const
}

export function subscriptionQuotaOptions(account: AccountItem) {
  const queryKey = subscriptionQuotaKey(account)
  const enabled =
    account.authMethod === "subscription_login" &&
    account.authState !== "logged_out" &&
    account.authState !== "reauth_required"
  return queryOptions({
    // 同一身份的元数据更新时间不拆分缓存；刷新和失败都保留最后成功的快照。
    queryKey,
    queryFn: async ({ signal, client }) => {
      if (!client.getQueryData(queryKey)) {
        try {
          // 本地恢复不占用官方查询队列；冷启动先显示旧快照，再等待网络结果。
          const cached = await readCachedSubscriptionQuota(account.id)
          if (cached && !signal.aborted && !client.getQueryData(queryKey)) {
            client.setQueryData<SubscriptionQuota>(queryKey, cached, {
              updatedAt: Date.parse(cached.fetchedAt) || 0,
            })
          }
        } catch {
          // 本地恢复失败仍尝试真实读取；该路径同时持久化，若存储仍失败会显示稳定错误。
        }
      }
      return queueSubscriptionQuery(() => readSubscriptionQuota(account.id), signal)
    },
    enabled,
    retry: false,
    staleTime: 60_000,
    gcTime: 10 * 60_000,
  })
}

export function useSubscriptionQuota(account: AccountItem) {
  const client = useQueryClient()
  const options = subscriptionQuotaOptions(account)
  const scope = JSON.stringify(options.queryKey)
  const previous = useRef({ scope, version: account.updatedAt })
  const query = useQuery(options)
  const refetch = query.refetch
  useEffect(
    () =>
      Events.On("subscription:quota-updated", () => {
        void client.invalidateQueries(
          { predicate: (query) => JSON.stringify(query.queryKey) === scope },
          { cancelRefetch: false },
        )
      }),
    [client, scope],
  )
  useEffect(() => {
    const changed =
      previous.current.scope === scope && previous.current.version !== account.updatedAt
    previous.current = { scope, version: account.updatedAt }
    if (changed && options.enabled) void refetch({ cancelRefetch: false })
  }, [account.updatedAt, scope, options.enabled, refetch])
  return query
}
