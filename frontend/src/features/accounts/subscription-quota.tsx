import { useEffect, useState } from "react"
import { RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import type { AccountItem } from "../../services/omniproxy"
import { subscriptionError, type SubscriptionQuotaWindow } from "../../services/subscription"
import { useSubscriptionQuota } from "./use-subscription-quota"
import { QuotaResetTime } from "./quota-reset-time"
import "./subscription-quota.css"

export function SubscriptionQuota({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const query = useSubscriptionQuota(account)
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 60_000)
    return () => clearInterval(timer)
  }, [])
  const authenticated =
    account.authState !== "logged_out" && account.authState !== "reauth_required"
  const fiveHour = query.data?.fiveHourEnabled ?? ["plus", "team"].includes(account.authPlan || "")
  return (
    <div
      aria-busy={authenticated && query.isFetching}
      className="bg-muted/20 min-w-0 space-y-2.5 rounded-xl border p-3"
    >
      {fiveHour ? (
        <SubscriptionQuotaRow
          label={t("subscription.quota.fiveHour")}
          window={authenticated ? query.data?.fiveHour : undefined}
          now={now}
          color="bg-blue-600"
          loading={authenticated && query.isPending}
        />
      ) : null}
      <SubscriptionQuotaRow
        label={t("subscription.quota.weekly")}
        window={authenticated ? query.data?.weekly : undefined}
        now={now}
        color="bg-emerald-500"
        loading={authenticated && query.isPending}
      />
      {!authenticated ? (
        <p className="text-muted-foreground text-xs">{t("subscription.quota.signIn")}</p>
      ) : query.isPending ? (
        <p role="status" className="sr-only">
          {t("subscription.quota.loading")}
        </p>
      ) : query.isError ? (
        <div className="flex items-center justify-between gap-2">
          <p role="alert" className="text-destructive text-xs">
            {t(`subscription.error.${subscriptionError(query.error)}`)}
          </p>
          <Tooltip content={t("subscription.quota.retry")}>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label={t("subscription.quota.retry")}
              disabled={query.isFetching}
              onClick={() => void query.refetch()}
            >
              <RefreshCw className="size-3" aria-hidden="true" />
            </Button>
          </Tooltip>
        </div>
      ) : !query.data?.weekly || (fiveHour && !query.data?.fiveHour) ? (
        <p className="text-muted-foreground text-[10px]">{t("subscription.quota.missingWindow")}</p>
      ) : null}
    </div>
  )
}

export function SubscriptionQuotaRow({
  label,
  window,
  now,
  color,
  loading,
}: {
  label: string
  window?: SubscriptionQuotaWindow | null
  now: number
  color: string
  loading: boolean
}) {
  const { t } = useTranslation()
  return (
    <div className="min-w-0 space-y-1.5">
      <div className="flex min-w-0 flex-wrap items-center justify-between gap-x-2 gap-y-1 text-xs">
        <div className="flex min-w-0 items-center gap-2">
          <span className="shrink-0 font-medium">{label}</span>
          {loading ? (
            <span aria-hidden="true" className="bg-muted inline-block h-4 w-20 rounded-full" />
          ) : (
            <QuotaResetTime window={window} now={now} />
          )}
        </div>
        <span className="min-w-[4ch] text-right font-mono text-[11px] tabular-nums">
          {loading ? (
            <span aria-hidden="true" className="bg-muted inline-block h-3 w-8 rounded" />
          ) : (
            <span key={window?.remainingPercent ?? "unknown"} className="quota-information-update">
              {window ? `${window.remainingPercent}%` : "—"}
            </span>
          )}
        </span>
      </div>
      <div
        role={window ? "progressbar" : undefined}
        aria-hidden={!window}
        aria-label={t("subscription.quota.remainingLabel", { label })}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={window?.remainingPercent}
        className="bg-muted h-1.5 overflow-hidden rounded-full"
      >
        <div
          className={`${color} h-full rounded-full transition-[width] duration-500 ease-out motion-reduce:transition-none`}
          style={{ width: `${window?.remainingPercent ?? 0}%` }}
        />
      </div>
    </div>
  )
}
