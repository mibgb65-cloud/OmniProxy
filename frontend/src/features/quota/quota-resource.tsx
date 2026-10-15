import { MiniMaxResource } from "../accounts/minimax-resource"
import { StepFunBalanceValues, StepFunPlanResource } from "../accounts/stepfun-resource"
import { useTranslation } from "react-i18next"
import { balanceError } from "../../services/balance"
import { subscriptionError } from "../../services/subscription"
import { SubscriptionQuotaRow } from "../accounts/subscription-quota"
import { quotaWindows, type QuotaEntry } from "./quota-model"

export function QuotaResource({ entry, now }: { entry: QuotaEntry; now: number }) {
  const { t } = useTranslation()
  if (entry.kind === "minimax_plan" || entry.provider?.kind === "minimax")
    return <MiniMaxResource account={entry.account} />
  if (entry.kind === "step_plan") return <StepFunPlanResource account={entry.account} />
  if (entry.kind === "unsupported" || !entry.canQuery) {
    return (
      <p className="text-muted-foreground text-xs leading-5">
        {t(entry.kind === "unsupported" ? "quota.unsupportedHint" : "quota.signInHint")}
      </p>
    )
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
    </div>
  )
}
