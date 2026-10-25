import { useEffect } from "react"
import { useQuery } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import { readAccountTokenUsage } from "../../services/activity"
import type { AccountItem } from "../../services/omniproxy"

// 本机留存用量：没有官方额度／余额接口的账号（River、自定义提供商等）用它展示已用 Token。
// 面板与「正在使用」指示共用同一查询键，事件刷新保持两者同步。
export function useAccountTokenUsage(
  account: AccountItem,
  { enabled = true }: { enabled?: boolean } = {},
) {
  const query = useQuery({
    queryKey: ["account-token-usage", account.id, account.updatedAt],
    queryFn: () => readAccountTokenUsage(account.id),
    retry: false,
    staleTime: 30_000,
    enabled,
  })
  const refetch = query.refetch
  useEffect(() => {
    if (!enabled) return
    const refresh = () => {
      if (document.visibilityState !== "hidden") void refetch({ cancelRefetch: false })
    }
    const off = Events.On("activity:updated", refresh)
    document.addEventListener("visibilitychange", refresh)
    return () => {
      off()
      document.removeEventListener("visibilitychange", refresh)
    }
  }, [enabled, refetch])
  return query
}
