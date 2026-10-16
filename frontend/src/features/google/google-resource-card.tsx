import { useTranslation } from "react-i18next"
import { useEffect, useState } from "react"
import { Card, CardContent } from "../../components/ui/card"
import type { GoogleResource } from "../../services/google"
import type { AccountItem } from "../../services/omniproxy"
import { accountDisplayName, formatAccountTime } from "../accounts/account-display"
import { GoogleProjectDialog } from "./google-project-dialog"

const categories = {
  rate_limited: "google.rateLimited",
  daily_exhausted: "google.dailyExhausted",
  payment_required: "google.paymentRequired",
  authentication_failed: "google.authenticationFailed",
  permission_denied: "google.permissionDenied",
} as const

export function GoogleResourceCard({
  resource,
  accounts,
}: {
  resource: GoogleResource
  accounts: AccountItem[]
}) {
  return (
    <Card className="min-w-0 gap-0 py-0">
      <CardContent className="p-4">
        <GoogleResourceContent resource={resource} accounts={accounts} />
      </CardContent>
    </Card>
  )
}

export function GoogleResourceContent({
  resource,
  accounts,
}: {
  resource: GoogleResource
  accounts: AccountItem[]
}) {
  const { t } = useTranslation()
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 30_000)
    return () => clearInterval(timer)
  }, [])
  const members = accounts.filter((account) => resource.accountIds?.includes(account.id))
  const observations = resource.observations ?? []
  return (
    <div className="min-w-0 space-y-3">
      <h3 className="font-semibold wrap-anywhere">{resource.projectId || t("google.unbound")}</h3>
      <p className="text-muted-foreground text-xs">
        {t(resource.projectId ? "google.shared" : "google.scopeUnknown")}
      </p>
      <dl className="text-sm">
        <dt className="text-muted-foreground">{t("google.officialRemaining")}</dt>
        <dd>{t("google.unknown")}</dd>
      </dl>
      <p className="text-muted-foreground text-xs">{t("google.localUsage")}</p>
      <ul className="space-y-2">
        {members.map((account) => (
          <li key={account.id} className="flex items-center justify-between gap-2 text-sm">
            <span className="min-w-0 wrap-anywhere">{accountDisplayName(account)}</span>
            <GoogleProjectDialog account={account} projectId={resource.projectId} />
          </li>
        ))}
      </ul>
      {observations.length ? (
        <ul className="space-y-3 border-t pt-3 text-xs">
          {observations.map((item) => (
            <li
              key={`${item.model}:${item.category}:${item.metric}`}
              className="space-y-1 wrap-anywhere"
            >
              <p className="font-medium">
                {item.model} ·{" "}
                {t(categories[item.category as keyof typeof categories] ?? "google.observed")}
              </p>
              <p className="text-muted-foreground">
                {t("google.observedAt", { time: formatAccountTime(item.observedAt) })}
              </p>
              {item.metric ? (
                <p>
                  {item.metric}
                  {item.limit != null ? ` · ${t("google.limit", { count: item.limit })}` : ""}
                </p>
              ) : null}
              {item.retryAt ? (
                <p>
                  {t(item.category === "daily_exhausted" ? "google.dailyReset" : "google.retryAt", {
                    time: formatAccountTime(item.retryAt),
                  })}
                </p>
              ) : null}
              {item.stale || (item.retryAt && Date.parse(item.retryAt) <= now) ? (
                <p className="text-muted-foreground">{t("google.stale")}</p>
              ) : null}
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-muted-foreground border-t pt-3 text-xs">{t("google.noObservations")}</p>
      )}
    </div>
  )
}
