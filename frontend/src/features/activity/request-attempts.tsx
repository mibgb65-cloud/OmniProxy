import { useTranslation } from "react-i18next"
import { formatTokens, type RequestItem } from "../../services/activity"
import { stateLabel, requestTime } from "./activity-format"
import { gatewayErrorKey } from "../gateway/gateway-errors"

export function RequestAttempts({ item }: { item: Pick<RequestItem, "attempts"> }) {
  const { t, i18n } = useTranslation()
  if (!item.attempts?.length) return null
  return (
    <section aria-label={t("failover.attempts")} className="space-y-3 border-t pt-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="text-sm font-semibold">{t("failover.attempts")}</h3>
        {item.attempts.length > 1 ? (
          <span className="text-muted-foreground text-xs">
            {t("failover.switched", { count: item.attempts.length - 1 })}
          </span>
        ) : null}
      </div>
      <ol className="space-y-2">
        {item.attempts.map((attempt) => (
          <li key={attempt.index} className="space-y-2 rounded-lg border p-3 text-xs">
            <div className="flex flex-wrap items-start justify-between gap-2">
              <p className="min-w-0 font-medium wrap-anywhere">
                {t("failover.attempt", {
                  index: attempt.index,
                  account: attempt.accountName || t("failover.account"),
                })}
              </p>
              <span>
                {t(stateLabel(attempt.state))} · {attempt.statusCode || "—"}
              </span>
            </div>
            <p className="text-muted-foreground wrap-anywhere">
              {attempt.providerName} · {attempt.model}
            </p>
            <p className="text-muted-foreground">
              {requestTime(attempt.startedAt)} · {attempt.latencyMs} ms
            </p>
            {attempt.errorCode ? (
              <p className="text-muted-foreground">{t(gatewayErrorKey(attempt.errorCode))}</p>
            ) : null}
            <p className="text-muted-foreground">
              {t("activity.input")}：
              {attempt.inputKnown
                ? formatTokens(attempt.inputTokens, i18n.resolvedLanguage)
                : t("activity.unknown")}{" "}
              · {t("activity.output")}：
              {attempt.outputKnown
                ? formatTokens(attempt.outputTokens, i18n.resolvedLanguage)
                : t("activity.unknown")}
            </p>
          </li>
        ))}
      </ol>
    </section>
  )
}
