import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import type { AccountItem } from "../../services/omniproxy"
import { readSubscriptionQuotaEstimate, type SubscriptionQuota } from "../../services/subscription"
import type { QuotaWindowEstimate } from "../../../bindings/omniproxy/internal/application/activity"
import type { TranslationKey } from "../../i18n/translation"
import { subscriptionQuotaKey } from "./use-subscription-quota"

const statusKeys: Record<string, TranslationKey> = {
  missing_window: "subscription.usd.missingWindow",
  expired: "subscription.usd.expired",
  insufficient_usage: "subscription.usd.insufficient",
  unknown_price: "subscription.usd.unknownPrice",
  missing_usage: "subscription.usd.missingUsage",
  no_usage: "subscription.usd.noUsage",
}
const currency = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" })

function usdRange(low: number, high: number) {
  if (!Number.isFinite(low) || !Number.isFinite(high) || low < 0 || high < low) return null
  const first = currency.format(low)
  const last = currency.format(high)
  return first === last ? first : `${first}–${last}`
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
    enabled: !!quota && !!account.authEmail && !!account.authAccountId,
    retry: false,
    staleTime: 60_000,
  })
  // 若并发刷新已写入更新的快照，旧百分比不能搭配新估值；等待同一快照的下一次读取。
  const estimate = query.data?.fetchedAt === quota?.fetchedAt ? query.data : undefined
  return (
    <section className="space-y-2 border-t pt-2.5" aria-label={t("subscription.usd.title")}>
      <p className="text-xs font-medium">{t("subscription.usd.title")}</p>
      <dl className="space-y-2 text-xs">
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
      <details className="text-muted-foreground text-[11px]">
        <summary className="cursor-pointer rounded-sm focus-visible:outline-2">
          {t("subscription.usd.scope")}
        </summary>
        <p className="mt-1 leading-relaxed">{t("subscription.usd.explanation")}</p>
      </details>
    </section>
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
  const total = estimate?.status === "ready" ? usdRange(estimate.totalMin, estimate.totalMax) : null
  const remaining =
    estimate?.status === "ready" ? usdRange(estimate.remainingMin, estimate.remainingMax) : null
  const key = expired
    ? "subscription.usd.expired"
    : error
      ? "subscription.usd.unavailable"
      : (statusKeys[estimate?.status ?? ""] ?? "subscription.usd.pending")
  return (
    <div className="flex min-w-0 flex-wrap items-start justify-between gap-x-2 gap-y-1">
      <dt className="shrink-0">{label}</dt>
      <dd className="min-w-0 text-right">
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
