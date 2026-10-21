import { useEffect } from "react"
import { useQuery } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import { useTranslation } from "react-i18next"
import type { AccountItem } from "../../services/omniproxy"
import { readSubscriptionQuotaEstimate, type SubscriptionQuota } from "../../services/subscription"
import type { QuotaWindowEstimate } from "../../../bindings/omniproxy/internal/application/activity"
import { subscriptionQuotaKey } from "./use-subscription-quota"
import { AccountTokenSummary } from "./account-token-summary"

const currency = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" })

function usd(value?: number) {
  return value !== undefined && Number.isFinite(value) && value >= 0 ? currency.format(value) : null
}

export function SubscriptionQuotaUSD({
  account,
  quota,
  fiveHour,
  now,
}: {
  account: AccountItem
  quota?: SubscriptionQuota
  fiveHour: boolean
  now: number
}) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: [
      ...subscriptionQuotaKey(account),
      "usd",
      quota?.fetchedAt ?? "",
      quota?.fiveHour?.resetsAt ?? "",
      quota?.weekly?.resetsAt ?? "",
      quota?.fiveHour?.remainingPercent,
      quota?.weekly?.remainingPercent,
    ],
    queryFn: () => readSubscriptionQuotaEstimate(account.id),
    enabled: !!account.authEmail && !!account.authAccountId,
    retry: false,
    staleTime: 60_000,
  })
  const refetch = query.refetch
  useEffect(() => {
    const refresh = () => {
      if (account.authEmail && account.authAccountId && document.visibilityState !== "hidden")
        void refetch({ cancelRefetch: false })
    }
    const off = Events.On("activity:updated", refresh)
    // 应用全局关闭了焦点重读；后台跳过统计后，恢复可见时补读，避免停留在旧用量。
    document.addEventListener("visibilitychange", refresh)
    return () => {
      off()
      document.removeEventListener("visibilitychange", refresh)
    }
  }, [refetch, account.authEmail, account.authAccountId])
  // 若并发刷新已写入更新的快照，旧百分比不能搭配新估值；等待同一快照的下一次读取。
  const estimate = query.data?.fetchedAt === quota?.fetchedAt ? query.data : undefined
  return (
    <>
      <section className="border-t pt-2" aria-label={t("subscription.usd.title")}>
        <table className="w-full table-fixed text-xs">
          <thead className="text-muted-foreground">
            <tr>
              <th scope="col" className="w-10 text-left font-normal">
                <span className="sr-only">{t("subscription.usd.window")}</span>
              </th>
              <th scope="col" className="pb-1 text-right font-normal">
                {t("subscription.usd.total")}
              </th>
              <th scope="col" className="pb-1 text-right font-normal">
                {t("subscription.usd.remaining")}
              </th>
            </tr>
          </thead>
          <tbody>
            {fiveHour ? (
              <EstimateRow
                label={t("subscription.usd.fiveHour")}
                estimate={estimate?.fiveHour}
                expired={!!quota?.fiveHour?.resetsAt && Date.parse(quota.fiveHour.resetsAt) <= now}
                error={query.isError}
              />
            ) : null}
            <EstimateRow
              label={t("subscription.usd.weekly")}
              estimate={estimate?.weekly}
              expired={!!quota?.weekly?.resetsAt && Date.parse(quota.weekly.resetsAt) <= now}
              error={query.isError}
            />
          </tbody>
        </table>
      </section>
      <AccountTokenSummary usage={query.data?.usage} error={query.isError} />
    </>
  )
}

function EstimateRow({
  label,
  estimate,
  expired,
  error,
}: {
  label: string
  estimate?: QuotaWindowEstimate | null
  expired: boolean
  error: boolean
}) {
  const ready = !expired && !error && estimate?.status === "ready"
  const total = ready ? usd(estimate.total) : null
  const remaining = ready ? usd(estimate.remaining) : null
  const available = total !== null && remaining !== null
  return (
    <tr>
      <th scope="row" className="text-muted-foreground py-0.5 text-left font-normal">
        {label}
      </th>
      <td className="py-0.5 text-right font-mono break-all tabular-nums">
        {available ? `≈ ${total}` : "—"}
      </td>
      <td className="py-0.5 text-right font-mono break-all tabular-nums">
        {available ? `≈ ${remaining}` : "—"}
      </td>
    </tr>
  )
}
