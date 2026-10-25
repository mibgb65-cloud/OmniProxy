import { MiMoPlanResource } from "../accounts/mimo-resource"
import { KimiCodeResource } from "../accounts/kimi-code-resource"
import { MiniMaxResource } from "../accounts/minimax-resource"
import { KimiBalanceValues } from "../accounts/kimi-balance"
import { StepFunBalanceValues, StepFunPlanResource } from "../accounts/stepfun-resource"
import { useTranslation } from "react-i18next"
import { balanceError } from "../../services/balance"
import { subscriptionError } from "../../services/subscription"
import { SubscriptionQuotaRow } from "../accounts/subscription-quota"
import { AccountUsagePanel } from "../accounts/account-usage-panel"
import { quotaWindows, type QuotaEntry } from "./quota-model"
import { GoogleQuotaValues } from "../google/google-quota-values"
import { GoogleSubscriptionModels } from "../google/google-subscription-models"

export function QuotaResource({ entry, now }: { entry: QuotaEntry; now: number }) {
  const { t } = useTranslation()
  if (entry.kind === "google_subscription")
    return (
      <div className="bg-muted/20 space-y-3 rounded-xl border p-3" aria-busy={entry.fetching}>
        <GoogleSubscriptionModels account={entry.account} />
        {entry.canQuery ? (
          <GoogleQuotaValues quota={entry.googleQuota} loading={entry.pending} now={now} />
        ) : (
          <p className="text-muted-foreground text-xs">{t("quota.signInHint")}</p>
        )}
        {entry.error ? (
          <p role="alert" className="text-destructive text-xs">
            {t(`subscription.error.${subscriptionError(entry.error)}`)}
          </p>
        ) : null}
        <p className="text-muted-foreground text-[10px] leading-5">{t("google.quota.hint")}</p>
        {entry.canQuery && entry.googleQuota?.fromCache ? (
          <p className="text-muted-foreground text-[10px]">{t("subscription.quota.retained")}</p>
        ) : null}
      </div>
    )
  if (entry.kind === "kimi_coding") return <KimiCodeResource account={entry.account} />
  if (entry.kind === "minimax_plan" || entry.provider?.kind === "minimax")
    return <MiniMaxResource account={entry.account} />
  if (entry.kind === "mimo_plan") return <MiMoPlanResource account={entry.account} />
  if (entry.kind === "step_plan") return <StepFunPlanResource account={entry.account} />
  if (entry.kind === "unsupported") {
    return (
      <div className="bg-muted/20 space-y-3 rounded-xl border p-3">
        <p className="text-muted-foreground text-xs leading-5">{t("quota.unsupportedHint")}</p>
        <AccountUsagePanel account={entry.account} />
        <p className="text-muted-foreground text-[10px] leading-5">
          {t("quota.unsupportedUsageNote")}
        </p>
      </div>
    )
  }
  if (!entry.canQuery) {
    return <p className="text-muted-foreground text-xs leading-5">{t("quota.signInHint")}</p>
  }
  return (
    <div className="bg-muted/20 space-y-3 rounded-xl border p-3" aria-busy={entry.fetching}>
      {entry.kind === "subscription" ? (
        <div className="space-y-2.5">
          {quotaWindows(entry.account, entry.quota).map(({ label, window, color }) => (
            <SubscriptionQuotaRow
              key={label}
              label={t(label)}
              window={window}
              now={now}
              color={color}
              loading={entry.pending && !entry.quota}
            />
          ))}
        </div>
      ) : entry.balance?.stepfun ? (
        <StepFunBalanceValues value={entry.balance.stepfun} />
      ) : entry.balance?.kimi ? (
        <KimiBalanceValues value={entry.balance.kimi} />
      ) : entry.balance ? (
        <div className="space-y-3">
          {(entry.balance.entries ?? []).map((item) => (
            <div key={item.currency} className="space-y-2">
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <span className="text-muted-foreground text-xs">
                  {t("balance.total")} · {item.currency}
                </span>
                <span className="quota-information-update max-w-full font-mono text-lg font-semibold break-all tabular-nums">
                  {item.total}
                </span>
              </div>
              <dl className="text-muted-foreground grid grid-cols-2 gap-3 text-xs">
                <div>
                  <dt>{t("balance.granted")}</dt>
                  <dd className="mt-1 font-mono break-all tabular-nums">{item.granted}</dd>
                </div>
                <div>
                  <dt>{t("balance.toppedUp")}</dt>
                  <dd className="mt-1 font-mono break-all tabular-nums">{item.toppedUp}</dd>
                </div>
              </dl>
            </div>
          ))}
        </div>
      ) : entry.pending ? (
        <div aria-hidden="true" className="space-y-2">
          <div className="bg-muted h-6 w-28 rounded" />
          <div className="bg-muted h-4 w-36 max-w-full rounded" />
        </div>
      ) : null}
      {entry.error ? (
        <p role="alert" className="text-destructive text-xs leading-5">
          {entry.kind === "subscription"
            ? t(`subscription.error.${subscriptionError(entry.error)}`)
            : t(`balance.error.${balanceError(entry.error)}`, {
                defaultValue: t("balance.error.query_failed"),
              })}
        </p>
      ) : entry.status === "unknown" || entry.status === "stale" ? (
        <p className="text-muted-foreground text-xs leading-5">
          {t(entry.status === "stale" ? "quota.staleHint" : "quota.missingHint")}
        </p>
      ) : null}
      {entry.kind === "subscription" && entry.quota?.fromCache ? (
        <p className="text-muted-foreground text-[10px]">{t("subscription.quota.retained")}</p>
      ) : null}
    </div>
  )
}
