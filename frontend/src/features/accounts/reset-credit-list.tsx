import { TicketCheck } from "lucide-react"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { SubscriptionResetCredit } from "../../services/subscription"
import { formatAccountTime } from "./account-display"

export function ResetCreditList({
  cards,
  busy,
  onSelect,
}: {
  cards: SubscriptionResetCredit[]
  busy: boolean
  onSelect: (id: string) => void
}) {
  const { t } = useTranslation()
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 60_000)
    return () => clearInterval(timer)
  }, [])
  return (
    <ul className="space-y-3">
      {cards.map((card) => {
        const expired = !!card.expiresAt && Date.parse(card.expiresAt) <= now
        const available =
          card.status === "available" && card.resetType === "codexRateLimits" && !expired
        const knownStatus =
          card.status === "available" || card.status === "redeeming" || card.status === "redeemed"
            ? card.status
            : "unknown"
        return (
          <li key={card.id} className="bg-muted/20 space-y-2 rounded-xl border p-3">
            <div className="flex items-start justify-between gap-3">
              <div className="flex min-w-0 items-center gap-2">
                <TicketCheck className="text-primary size-4 shrink-0" aria-hidden="true" />
                <span className="min-w-0 text-sm font-medium break-words">
                  {card.title || t("subscription.reset.cardTitle")}
                </span>
              </div>
              <span className="bg-muted text-muted-foreground shrink-0 rounded-md px-2 py-1 text-xs">
                {expired
                  ? t("subscription.reset.expired")
                  : t(`subscription.reset.status.${knownStatus}`)}
              </span>
            </div>
            {card.description ? (
              <p className="text-muted-foreground text-sm break-words">{card.description}</p>
            ) : null}
            <div className="flex flex-wrap items-end justify-between gap-3">
              <dl className="text-muted-foreground space-y-1 text-xs">
                <div>
                  <dt className="inline">{t("subscription.reset.granted")}</dt>{" "}
                  <dd className="inline">{formatAccountTime(card.grantedAt)}</dd>
                </div>
                <div>
                  <dt className="inline">{t("subscription.reset.expires")}</dt>{" "}
                  <dd className="inline">
                    {card.expiresAt
                      ? formatAccountTime(card.expiresAt)
                      : t("subscription.reset.noExpiry")}
                  </dd>
                </div>
              </dl>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={busy || !available}
                onClick={() => onSelect(card.id)}
              >
                {t("subscription.reset.use")}
              </Button>
            </div>
          </li>
        )
      })}
    </ul>
  )
}
