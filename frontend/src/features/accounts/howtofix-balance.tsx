import { useTranslation } from "react-i18next"
import type { Balance as HowToFixBalance } from "../../../bindings/omniproxy/internal/provider/howtofix/models"

// 上游未返回的金额保持空字符串，界面显示为未知而不是零，避免把缺数据误报成余额耗尽。
export function HowToFixBalanceValues({ value }: { value: HowToFixBalance }) {
  const { t } = useTranslation()
  const amountPrefix = value.currency === "CNY" ? "¥" : "US$"
  return (
    <div className="min-w-0 space-y-2">
      <div className="flex min-w-0 flex-wrap items-baseline justify-between gap-2">
        <span className="text-muted-foreground text-xs">
          {t("balance.remaining")} <span className="font-mono">{value.currency}</span>
        </span>
        <span
          className="quota-information-update max-w-full font-mono text-xl font-semibold break-all tabular-nums"
          key={`${value.currency}:${value.remaining}`}
        >
          {value.remaining === "" ? "—" : `${amountPrefix}${value.remaining}`}
        </span>
      </div>
    </div>
  )
}
