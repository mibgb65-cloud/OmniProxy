import { useQuery } from "@tanstack/react-query"
import type { AccountItem } from "../../services/omniproxy"
import { readGoogleResources } from "../../services/google"

export function useGoogleResources(accounts: AccountItem[]) {
  // 共享项目只查询一次；更换任一 Key 后不展示旧凭证的绑定和观察。
  const version = accounts
    .map((account) => `${account.id}:${account.updatedAt}`)
    .sort()
    .join("|")
  return useQuery({
    queryKey: ["google-resources", version],
    queryFn: readGoogleResources,
    enabled: accounts.length > 0,
    staleTime: 30_000,
    refetchInterval: 30_000,
  })
}
