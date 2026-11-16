import { useTranslation } from "react-i18next"
import { AccountUsagePanel } from "../accounts/account-usage-panel"
import { formatAccountTime } from "../accounts/account-display"
import { dumateErrorCode, dumateReasonKey } from "../dumate/dumate-error"
import { formatPoints } from "../dumate/dumate-format"
import type { QuotaEntry } from "./quota-model"

/** DuMate 积分来自官方控制台，本机 Token 用量来自本机请求记录；两者不换算也不相加。 */
export function DuMateQuotaResource({ entry }: { entry: QuotaEntry }) {
  const { t } = useTranslation()
  const data = entry.dumate
  const points = data?.points
  const reason = (code?: string) => t(dumateReasonKey(code))

  return (
    <div className="bg-muted/20 space-y-3 rounded-xl border p-3" aria-busy={entry.fetching}>
      {points ? (
        <div className="space-y-2">
          <div className="grid grid-cols-2 gap-2 text-xs">
            <div>
              <p className="text-muted-foreground">{t("dumate.points.remaining")}</p>
              <p className="quota-information-update mt-0.5 font-mono text-lg font-semibold break-all tabular-nums">
                {t("dumate.points.remainingValue", {
                  points: formatPoints(points.remainingPoints),
                })}
              </p>
            </div>
            <div>
              <p className="text-muted-foreground">{t("dumate.points.used")}</p>
              <p className="mt-0.5 font-mono text-sm break-all tabular-nums">
                {t("dumate.points.usage", {
                  used: formatPoints(points.usedPoints),
                  total: formatPoints(points.totalPoints),
                })}
              </p>
            </div>
          </div>
          {points.expiringAt ? (
            <p className="text-muted-foreground text-[11px] leading-5">
              {t("dumate.points.expiring", {
                points: formatPoints(points.expiringPoints ?? 0),
                time: formatAccountTime(points.expiringAt),
              })}
            </p>
          ) : null}
        </div>
      ) : entry.pending ? (
        <div aria-hidden="true" className="space-y-2">
          <div className="bg-muted h-6 w-28 rounded" />
          <div className="bg-muted h-4 w-36 max-w-full rounded" />
        </div>
      ) : null}
      {entry.error ? (
        <p role="alert" className="text-destructive text-xs leading-5">
          {t("dumate.points.unavailable", { reason: reason(dumateErrorCode(entry.error)) })}
        </p>
      ) : data?.state === "unavailable" ? (
        <p role="alert" className="text-destructive text-xs leading-5">
          {t("dumate.points.unavailable", { reason: reason(data.errorCode) })}
        </p>
      ) : data?.state === "stale" ? (
        <p role="alert" className="text-destructive text-xs leading-5">
          {t("dumate.points.stale", {
            time: data.fetchedAt ? formatAccountTime(data.fetchedAt) : "—",
            reason: reason(data.errorCode),
          })}
        </p>
      ) : !entry.canQuery ? (
        <p className="text-muted-foreground text-xs leading-5">{t("quota.signInHint")}</p>
      ) : null}
      <div className="space-y-2">
        <AccountUsagePanel account={entry.account} />
        <p className="text-muted-foreground text-[10px] leading-5">
          {t("quota.unsupportedUsageNote")}
        </p>
      </div>
    </div>
  )
}
