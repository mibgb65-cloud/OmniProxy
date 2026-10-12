import { useTranslation } from "react-i18next"
import { Tooltip } from "../../components/ui/tooltip"

export function SubscriptionAccountID({ id }: { id?: string }) {
  const { t } = useTranslation()
  const value = id || ""
  const preview = value.length > 20 ? `${value.slice(0, 8)}…${value.slice(-6)}` : value
  return (
    <Tooltip content={value ? `account_id: ${value}` : t("subscription.accountIdPending")}>
      <span
        tabIndex={0}
        className="text-muted-foreground focus-visible:ring-ring/50 min-w-0 truncate rounded-md font-mono text-[11px] outline-none focus-visible:ring-2"
      >
        account_id: {preview || t("subscription.accountIdUnknown")}
      </span>
    </Tooltip>
  )
}
