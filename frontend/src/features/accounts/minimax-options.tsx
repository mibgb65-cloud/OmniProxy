import { useTranslation } from "react-i18next"
import { SelectField } from "../../components/ui/select-field"
import type { MiniMaxOptions as Options } from "../../services/minimax"
import type { AccountItem } from "../../services/omniproxy"

export function MiniMaxOptionsFields({
  value,
  onChange,
  disabled,
  invalid = false,
}: {
  value: Options
  onChange: (value: Options) => void
  disabled: boolean
  invalid?: boolean
}) {
  const { t } = useTranslation()
  return (
    <fieldset className="space-y-3" disabled={disabled} aria-describedby="minimax-key-hint">
      <legend className="mb-2 text-sm font-medium">{t("minimax.connection")}</legend>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <label htmlFor="minimax-region" className="text-xs">
            {t("minimax.region")}
          </label>
          <SelectField
            id="minimax-region"
            value={value.region}
            disabled={disabled}
            aria-invalid={invalid}
            aria-describedby={
              invalid ? "minimax-key-hint minimax-options-error" : "minimax-key-hint"
            }
            options={[
              { value: "", label: t("minimax.choose") },
              { value: "cn", label: t("minimax.cn") },
              { value: "global", label: t("minimax.global") },
            ]}
            onValueChange={(region) => onChange({ ...value, region })}
          />
        </div>
        <div className="space-y-1.5">
          <label htmlFor="minimax-billing" className="text-xs">
            {t("minimax.billing")}
          </label>
          <SelectField
            id="minimax-billing"
            value={value.billingMode}
            disabled={disabled}
            aria-invalid={invalid}
            aria-describedby={
              invalid ? "minimax-key-hint minimax-options-error" : "minimax-key-hint"
            }
            options={[
              { value: "", label: t("minimax.choose") },
              { value: "payg", label: t("minimax.payg") },
              { value: "subscription", label: t("minimax.plan") },
            ]}
            onValueChange={(billingMode) => onChange({ ...value, billingMode })}
          />
        </div>
      </div>
      <p id="minimax-key-hint" className="text-muted-foreground text-xs leading-5">
        {t("minimax.keyHint")}
      </p>
      {invalid ? (
        <p id="minimax-options-error" role="alert" className="text-destructive text-xs">
          {t("minimax.choose")}
        </p>
      ) : null}
    </fieldset>
  )
}

export function MiniMaxBadge({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const value = account.minimax
  if (!value && account.providerId !== "builtin_minimax") return null
  const configured =
    !!value &&
    ["cn", "global"].includes(value.region) &&
    ["payg", "subscription"].includes(value.billingMode)
  return (
    <span className="bg-muted text-muted-foreground inline-flex rounded-md px-2 py-0.5 text-[11px]">
      {configured && value
        ? `${t(value.region === "cn" ? "minimax.cn" : "minimax.global")} · ${t(value.billingMode === "payg" ? "minimax.payg" : "minimax.plan")}`
        : t("minimax.needsConfiguration")}
    </span>
  )
}
