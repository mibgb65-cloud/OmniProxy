import { CalendarDays, Clock3 } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Tooltip } from "../../components/ui/tooltip"
import { formatAccountTime } from "./account-display"
import type { SubscriptionQuotaWindow } from "../../services/subscription"

export function QuotaResetTime({
  window,
  now,
}: {
  window?: SubscriptionQuotaWindow | null
  now: number
}) {
  const { t } = useTranslation()
  const resetsAt = window?.resetsAt
  const reset = resetsAt ? Date.parse(resetsAt) : NaN
  let label: string = t("subscription.quota.timeUnknown")
  let detail: string = t("subscription.quota.timeUnknownHint")
  if (Number.isFinite(reset)) {
    const minutes = Math.ceil((reset - now) / 60_000)
    if (minutes <= 0) {
      label = t("subscription.quota.expired")
    } else if (minutes >= 1440) {
      label = t("subscription.quota.daysHours", {
        days: Math.floor(minutes / 1440),
        hours: Math.floor((minutes % 1440) / 60),
      })
    } else if (minutes >= 60) {
      label = t("subscription.quota.hoursMinutes", {
        hours: Math.floor(minutes / 60),
        minutes: minutes % 60,
      })
    } else {
      label = t("subscription.quota.minutes", { minutes })
    }
    detail = t("subscription.quota.resetAt", { time: formatAccountTime(resetsAt || "") })
  } else if (window?.remainingPercent === 100) {
    // 没有重置目标时本地倒计时尚未开始；提示中明确官方时间暂未返回。
    label = t("subscription.quota.notStarted")
  }
  const Icon = Number.isFinite(reset) ? CalendarDays : Clock3
  return (
    <Tooltip content={detail}>
      <span
        tabIndex={0}
        className="bg-muted text-muted-foreground focus-visible:ring-ring/50 inline-flex max-w-full items-center gap-1 rounded-full px-1.5 py-0.5 text-[10px] outline-none focus-visible:ring-2"
      >
        <Icon className="size-3 shrink-0" aria-hidden="true" />
        <span className="truncate">
          <span key={label} className="quota-information-update">
            {label}
          </span>
        </span>
      </span>
    </Tooltip>
  )
}
