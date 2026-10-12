import { queryOptions, useQuery } from "@tanstack/react-query"
import { useEffect, useRef } from "react"
import type { AccountItem } from "../../services/omniproxy"
import { readSubscriptionQuota } from "../../services/subscription"
import { queueSubscriptionQuery } from "../../lib/query-queue"

export function subscriptionQuotaKey(account: AccountItem) {
  return [
    "subscription-quota",
    account.id,
    account.authAccountId || account.authEmail || "",
    account.authState || "unknown",
  ] as const
}

export function subscriptionQuotaOptions(account: AccountItem) {
  const enabled =
    account.authMethod === "subscription_login" &&
    account.authState !== "logged_out" &&
    account.authState !== "reauth_required"
  return queryOptions({
    // 同一身份的元数据更新时间不拆分缓存；刷新和失败都保留最后成功的快照。
    queryKey: subscriptionQuotaKey(account),
    queryFn: ({ signal }) =>
      queueSubscriptionQuery(() => readSubscriptionQuota(account.id), signal),
    enabled,
    retry: false,
    staleTime: 60_000,
    gcTime: 10 * 60_000,
  })
}

export function useSubscriptionQuota(account: AccountItem) {
  const options = subscriptionQuotaOptions(account)
  const scope = JSON.stringify(options.queryKey)
  const previous = useRef({ scope, version: account.updatedAt })
  const query = useQuery(options)
  const refetch = query.refetch
  useEffect(() => {
    const changed =
      previous.current.scope === scope && previous.current.version !== account.updatedAt
    previous.current = { scope, version: account.updatedAt }
    if (changed && options.enabled) void refetch({ cancelRefetch: false })
  }, [account.updatedAt, scope, options.enabled, refetch])
  return query
}
