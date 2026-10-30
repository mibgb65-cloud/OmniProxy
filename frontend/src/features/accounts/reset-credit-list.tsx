import { TicketCheck } from "lucide-react"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { SubscriptionResetCredit } from "../../services/subscription"
import { formatAccountTime } from "./account-display"

/**
 * 官方重置卡类型枚举在不同环境下写法不一致（大小写、下划线）。
 * 只有明显属于别的权益（例如促销额度）才拒绝，其余按官方状态与有效期判定。
 */
export function usableResetType(value: string) {
  const normalized = value.toLowerCase().replace(/[\s_-]/g, "")
  return (
    normalized === "" ||
    normalized === "codexratelimits" ||
    normalized === "ratelimits" ||
    normalized === "ratelimit"
  )
}

/** 卡片当前是否可用于兑换：官方状态 + 类型写法 + 有效期。 */
export function resetCreditUsable(
  card: Pick<SubscriptionResetCredit, "status" | "resetType">,
  expired: boolean,
) {
  return card.status === "available" && usableResetType(card.resetType) && !expired
}

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
        // 徽标与按钮共用同一判定，避免出现「写着可使用、按钮是灰的」。
        const available = resetCreditUsable(card, expired)
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
                  : card.status === "available" && !available
                    ? t("subscription.reset.status.unusable")
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
