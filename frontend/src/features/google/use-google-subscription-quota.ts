import { queryOptions, useQuery } from "@tanstack/react-query"
import { useEffect, useRef } from "react"
import type { AccountItem } from "../../services/omniproxy"
import {
  readGoogleSubscriptionQuota,
  readGoogleSubscriptionModels,
} from "../../services/google-subscription"
import { queueSubscriptionQuery } from "../../lib/query-queue"

export function googleSubscriptionQuotaOptions(account: AccountItem) {
  return queryOptions({
    queryKey: [
      "google-subscription-quota",
      account.id,
      account.authAccountId || account.authEmail || "",
      account.authState || "unknown",
    ],
    queryFn: ({ signal }) =>
      queueSubscriptionQuery(() => readGoogleSubscriptionQuota(account.id), signal),
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
