import { useEffect } from "react"
import { useQuery } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import { useTranslation } from "react-i18next"
import { Tooltip } from "../../components/ui/tooltip"
import type { AccountItem } from "../../services/omniproxy"
import { readSubscriptionQuotaEstimate, type SubscriptionQuota } from "../../services/subscription"
import type { QuotaWindowEstimate } from "../../../bindings/omniproxy/internal/application/activity"
import type { TranslationKey } from "../../i18n/translation"
import { subscriptionQuotaKey } from "./use-subscription-quota"
import { AccountTokenSummary } from "./account-token-summary"

const statusKeys: Record<string, TranslationKey> = {
  missing_window: "subscription.usd.missingWindow",
  expired: "subscription.usd.expired",
  insufficient_usage: "subscription.usd.insufficient",
  unknown_price: "subscription.usd.unknownPrice",
  missing_usage: "subscription.usd.missingUsage",
  no_usage: "subscription.usd.noUsage",
}
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
      <section className="space-y-2 border-t pt-2.5" aria-label={t("subscription.usd.title")}>
        <Tooltip content={t("subscription.usd.explanation")}>
          <p tabIndex={0} className="w-fit rounded-sm text-xs font-medium focus-visible:outline-2">
            {t("subscription.usd.title")}
          </p>
        </Tooltip>
        <dl className={`grid min-w-0 gap-3 text-xs ${fiveHour ? "grid-cols-2" : "grid-cols-1"}`}>
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
        </dl>
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
  const { t } = useTranslation()
  const total = estimate?.status === "ready" ? usd(estimate.total) : null
  const remaining = estimate?.status === "ready" ? usd(estimate.remaining) : null
  const key = expired
    ? "subscription.usd.expired"
    : error
      ? "subscription.usd.unavailable"
      : (statusKeys[estimate?.status ?? ""] ?? "subscription.usd.pending")
  return (
    <div className="min-w-0 space-y-1">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="min-w-0 space-y-0.5 break-words">
        {!expired && !error && total && remaining ? (
          <>
            <p className="font-mono tabular-nums">≈ {total}</p>
            <p className="text-muted-foreground">
              {t("subscription.usd.remaining", { amount: remaining })}
            </p>
            {estimate?.resetAdjusted ? (
              <p className="text-muted-foreground">{t("subscription.usd.resetAdjusted")}</p>
            ) : null}
          </>
        ) : (
          <span className="text-muted-foreground">
            {t(key, { count: estimate?.unpricedRequests ?? 0 })}
          </span>
        )}
      </dd>
    </div>
  )
}
