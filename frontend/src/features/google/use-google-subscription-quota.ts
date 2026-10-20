import { queryOptions, useQuery } from "@tanstack/react-query"
import { useEffect, useRef } from "react"
import type { AccountItem } from "../../services/omniproxy"
import {
  readGoogleSubscriptionQuota,
  readCachedGoogleSubscriptionQuota,
  readGoogleSubscriptionModels,
  type GoogleSubscriptionQuota,
} from "../../services/google-subscription"
import { queueSubscriptionQuery } from "../../lib/query-queue"

export function googleSubscriptionQuotaOptions(account: AccountItem) {
  const queryKey = [
    "google-subscription-quota",
    account.id,
    account.authAccountId || "",
    account.authEmail?.trim().toLowerCase() || "",
    account.authState || "unknown",
  ] as const
  return queryOptions({
    queryKey,
    queryFn: async ({ signal, client }) => {
      if (!client.getQueryData(queryKey)) {
        try {
          const cached = await readCachedGoogleSubscriptionQuota(account.id)
          if (cached && !signal.aborted && !client.getQueryData(queryKey))
            client.setQueryData<GoogleSubscriptionQuota>(queryKey, cached, {
              updatedAt: Date.parse(cached.fetchedAt) || 0,
            })
        } catch {
          // 快照读取失败仍尝试联网；如果持久化仍失败，后端会返回稳定错误。
        }
      }
      return queueSubscriptionQuery(() => readGoogleSubscriptionQuota(account.id), signal)
    },
    enabled: account.authMethod === "subscription_login" && account.authState === "authenticated",
    retry: false,
    refetchOnWindowFocus: false,
    staleTime: 60_000,
    gcTime: 10 * 60_000,
  })
}

export function useGoogleSubscriptionQuota(account: AccountItem) {
  const options = googleSubscriptionQuotaOptions(account)
  const query = useQuery(options)
  const enabled =
    account.authMethod === "subscription_login" && account.authState === "authenticated"
  useMetadataRefresh(account, JSON.stringify(options.queryKey), enabled, query.refetch)
  return query
}

export function useGoogleSubscriptionModels(account: AccountItem) {
  const enabled =
    account.authMethod === "subscription_login" && account.authState === "authenticated"
  const query = useQuery({
    queryKey: [
      "google-subscription-models",
      account.id,
      account.authAccountId || account.authEmail || "",
      account.authState || "unknown",
    ],
    queryFn: ({ signal }) =>
      queueSubscriptionQuery(() => readGoogleSubscriptionModels(account.id), signal),
    enabled,
    retry: false,
    refetchOnWindowFocus: false,
    staleTime: 60_000,
    gcTime: 10 * 60_000,
  })
  const scope = JSON.stringify([
    account.id,
    account.authAccountId || account.authEmail || "",
    account.authState,
  ])
  useMetadataRefresh(account, scope, enabled, query.refetch)
  return query
}

function useMetadataRefresh(
  account: AccountItem,
  scope: string,
  enabled: boolean,
  refetch: (options?: { cancelRefetch?: boolean }) => unknown,
) {
  const previous = useRef({ scope, version: account.updatedAt })
  useEffect(() => {
    const changed =
      previous.current.scope === scope && previous.current.version !== account.updatedAt
    previous.current = { scope, version: account.updatedAt }
    if (changed && enabled) void refetch({ cancelRefetch: false })
  }, [account.updatedAt, scope, enabled, refetch])
}
