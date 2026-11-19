import { useQueries } from "@tanstack/react-query"
import type { AccountItem } from "../../services/omniproxy"
import type { GoogleSubscriptionQuota } from "../../services/google-subscription"
import { buildQuotaGroupSections, type QuotaGroupSection } from "./google-quota-groups"
import { googleSubscriptionQuotaOptions } from "./use-google-subscription-quota"

/**
 * 为账号页按额度分组分区。查询键与卡片内的额度查询一致，react-query 会共用同一次读取，
 * 分区本身不额外请求上游；未登录、尚未刷新或读取失败的订阅账号归入未知分区。
 */
export function useGoogleQuotaGroupSections(
  accounts: AccountItem[],
  enabled: boolean,
): QuotaGroupSection[] {
  const subscriptions = accounts.filter((account) => account.authMethod === "subscription_login")
  const queries = useQueries({
    queries: enabled ? subscriptions.map((account) => googleSubscriptionQuotaOptions(account)) : [],
  })
  const quotas = new Map<string, GoogleSubscriptionQuota | undefined>()
  if (enabled)
    subscriptions.forEach((account, index) => quotas.set(account.id, queries[index]?.data))
  return buildQuotaGroupSections(accounts, quotas)
}
