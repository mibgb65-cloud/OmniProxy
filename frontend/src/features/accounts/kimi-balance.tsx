import { useTranslation } from "react-i18next"
import type { KimiBalance } from "../../services/kimi"

export function KimiBalanceValues({ value }: { value: KimiBalance }) {
  const { t } = useTranslation()
  return (
    <div className="min-w-0 space-y-2">
      <div className="flex min-w-0 flex-wrap items-baseline justify-between gap-2">
        <span className="text-muted-foreground text-xs">
          {t("balance.availableAmount")} <span className="font-mono">{value.currency}</span>
        </span>
        <span
          className="quota-information-update max-w-full font-mono text-xl font-semibold break-all tabular-nums"
          key={`${value.currency}:${value.available}`}
        >
          {value.currency === "CNY" ? "¥" : "US$"}
          {value.available}
        </span>
      </div>
      <dl className="text-muted-foreground grid grid-cols-2 gap-x-3 gap-y-1 text-[11px]">
        <div>
          <dt>{t("balance.voucher")}</dt>
          <dd className="mt-0.5 font-mono break-all tabular-nums">{value.voucher}</dd>
        </div>
        <div>
          <dt>{t("balance.cash")}</dt>
          <dd className="mt-0.5 font-mono break-all tabular-nums">{value.cash}</dd>
        </div>
      </dl>
    </div>
  )
}
