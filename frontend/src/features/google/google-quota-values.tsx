import { useTranslation } from "react-i18next"
import type { GoogleQuotaBucket, GoogleSubscriptionQuota } from "../../services/google-subscription"
import { QuotaResetTime } from "../accounts/quota-reset-time"
import { googleQuotaPercent, googleQuotaState } from "./google-quota-model"

export function GoogleQuotaValues({
  quota,
  loading,
  now,
}: {
  quota?: GoogleSubscriptionQuota
  loading: boolean
  now: number
}) {
  const { t } = useTranslation()
  if (loading && !quota)
    return (
      <div role="status" className="space-y-3">
        <span className="sr-only">{t("google.quota.loading")}</span>
        <div aria-hidden="true" className="bg-muted h-5 w-28 rounded" />
        <div aria-hidden="true" className="bg-muted h-2 rounded" />
      </div>
    )
  if (!quota?.groups?.length)
    return <p className="text-muted-foreground text-xs leading-5">{t("google.quota.missing")}</p>
  return (
    <div className="space-y-4">
      {quota.groups.map((group) => {
        const name =
          group.name === "Gemini Models"
            ? t("google.quota.gemini")
            : group.name === "Claude and GPT models"
              ? t("google.quota.thirdParty")
              : group.name
        return (
          <section key={group.name} aria-label={name} className="min-w-0 space-y-2.5">
            <h4 className="text-xs font-medium wrap-anywhere">{name}</h4>
            {(group.buckets ?? []).map((bucket) => (
              <GoogleQuotaRow key={bucket.id} bucket={bucket} group={name} now={now} />
            ))}
            {!group.buckets?.length ? (
              <p className="text-muted-foreground text-xs">{t("google.quota.missing")}</p>
            ) : null}
          </section>
        )
      })}
      {googleQuotaState(quota, now) === "stale" ? (
        <p className="text-muted-foreground text-[11px] leading-5">{t("quota.staleHint")}</p>
      ) : null}
    </div>
  )
}

function GoogleQuotaRow({
  bucket,
  group,
  now,
}: {
  bucket: GoogleQuotaBucket
  group: string
  now: number
}) {
  const { t, i18n } = useTranslation()
  const label =
    bucket.window === "weekly"
      ? t("google.quota.weekly")
      : bucket.window === "5h"
        ? t("google.quota.fiveHour")
        : bucket.name || bucket.window || bucket.id
  const percent = bucket.remainingPercent
  const known = percent != null && Number.isFinite(percent) && percent >= 0 && percent <= 100
  return (
    <div className="min-w-0 space-y-1.5">
      <div className="flex flex-wrap items-center justify-between gap-2 text-xs">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <span>{label}</span>
          <QuotaResetTime
            window={{ remainingPercent: percent, resetsAt: bucket.resetsAt }}
            now={now}
            showNotStarted={false}
          />
        </div>
        <span className="font-mono text-[11px] tabular-nums">
          {googleQuotaPercent(percent, i18n.language)}
        </span>
      </div>
      <div
        role={known ? "progressbar" : undefined}
        aria-hidden={!known}
        aria-label={t("subscription.quota.remainingLabel", { label: `${group} · ${label}` })}
        aria-valuenow={known ? percent : undefined}
        aria-valuemin={0}
        aria-valuemax={100}
        className="bg-muted h-1.5 overflow-hidden rounded-full"
      >
        <div
          className={`${percent === 0 ? "bg-amber-500" : bucket.window === "5h" ? "bg-blue-600" : "bg-emerald-500"} h-full rounded-full transition-[width] duration-500 ease-out motion-reduce:transition-none`}
          style={{ width: `${known ? percent : 0}%` }}
        />
      </div>
    </div>
  )
}
