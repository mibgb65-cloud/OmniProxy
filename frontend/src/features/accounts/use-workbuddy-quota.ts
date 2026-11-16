import { queryOptions, useQuery } from "@tanstack/react-query"
import type { AccountItem } from "../../services/omniproxy"
import { workbuddyCachedQuota, workbuddyQuota, type WorkBuddyQuota } from "../../services/workbuddy"

// 账号元数据版本变化可能来自重新登录，旧凭据对应的额度不能继续用于新授权。
export function workbuddyQuotaKey(account: AccountItem) {
  return ["workbuddy-quota", account.id, account.updatedAt] as const
}

export function workbuddyQuotaOptions(account: AccountItem) {
  const queryKey = workbuddyQuotaKey(account)
  return queryOptions({
    queryKey,
    queryFn: async ({ signal, client }) => {
      if (!client.getQueryData(queryKey)) {
        try {
          // 本机保留的快照不占用上游请求；冷启动先显示它，再等待联网结果。
          const cached = await workbuddyCachedQuota(account.id)
          if (cached && !signal.aborted && !client.getQueryData(queryKey)) {
            client.setQueryData<WorkBuddyQuota>(queryKey, cached, {
              updatedAt: Date.parse(cached.fetchedAt) || 0,
            })
          }
        } catch {
          // 快照读取失败仍尝试真实读取；失败时会显示稳定错误码。
        }
      }
      return workbuddyQuota(account.id)
    },
    enabled: account.authMethod === "workbuddy_login",
    retry: false,
    staleTime: 60_000,
    gcTime: 10 * 60_000,
  })
}

export function useWorkBuddyQuota(account: AccountItem) {
  return useQuery(workbuddyQuotaOptions(account))
}
