import { useState } from "react"
import { useTranslation } from "react-i18next"
import { openStepConsole } from "../../services/stepfun"
import type { StepFunBalance as Balance } from "../../services/stepfun"
import type { AccountItem } from "../../services/omniproxy"
import { Button } from "../../components/ui/button"

export function StepFunBalanceValues({ value }: { value: Balance }) {
  const { t } = useTranslation()
  const currencyLabel = value.currency === "CNY" ? t("stepfun.cny") : t("stepfun.currencyUnknown")
  const amountPrefix = value.currency === "CNY" ? "¥" : ""
  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <span className="text-muted-foreground text-xs">{t("balance.total")}</span>
        <span className="font-mono text-xl font-semibold break-all tabular-nums">
          {amountPrefix}
          {value.amount}
        </span>
      </div>
      <p className="text-muted-foreground text-xs">
        {t(value.type === "prepaid" ? "stepfun.prepaid" : "stepfun.postpaid")} · {currencyLabel}
      </p>
      <dl className="text-muted-foreground grid grid-cols-2 gap-3 text-[11px]">
        <div>
          <dt>{t("stepfun.totalCash")}</dt>
          <dd className="mt-1 font-mono break-all">
            {amountPrefix}
            {value.totalCash}
          </dd>
        </div>
        <div>
          <dt>{t("stepfun.totalVoucher")}</dt>
          <dd className="mt-1 font-mono break-all">
            {amountPrefix}
            {value.totalVoucher}
          </dd>
        </div>
      </dl>
    </div>
  )
}

export function StepFunPlanResource({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const [failed, setFailed] = useState(false)
  const [opening, setOpening] = useState(false)
  async function open() {
    setOpening(true)
    setFailed(false)
    try {
      await openStepConsole(account.id)
    } catch {
      setFailed(true)
    } finally {
      setOpening(false)
    }
  }
  return (
    <section
      className="bg-muted/20 space-y-2 rounded-xl border p-3"
      aria-label={t("stepfun.credits")}
    >
      <p className="text-sm font-medium">{t("stepfun.credits")}</p>
      <p className="text-muted-foreground text-xs leading-5">{t("stepfun.planUnknown")}</p>
      <Button
        size="sm"
        variant="outline"
        className="cursor-pointer"
        disabled={opening}
        onClick={() => void open()}
      >
        {t("stepfun.openPlatform")}
      </Button>
      {failed ? (
        <p role="alert" className="text-destructive text-xs">
          {t("stepfun.openFailed")}
        </p>
      ) : null}
    </section>
  )
}
