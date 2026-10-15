import { useTranslation } from "react-i18next"
import { SelectField } from "../../components/ui/select-field"
import {
  MIMO_PLAN_REGIONS,
  miMoEndpoint,
  validMiMoOptions,
  type MiMoOptions,
} from "../../services/mimo"
import type { AccountItem } from "../../services/omniproxy"

export function MiMoOptionsFields({
  value,
  onChange,
  disabled,
  invalid = false,
}: {
  value: MiMoOptions
  onChange: (value: MiMoOptions) => void
  disabled: boolean
  invalid?: boolean
}) {
  const { t } = useTranslation()
  return (
    <fieldset className="space-y-3" disabled={disabled} aria-describedby="mimo-key-hint">
      <legend className="mb-2 text-sm font-medium">{t("mimo.connection")}</legend>
      <div className="space-y-1.5">
        <label htmlFor="mimo-billing" className="text-xs">
          {t("mimo.billing")}
        </label>
        <SelectField
          id="mimo-billing"
          value={value.billingMode}
          disabled={disabled}
          aria-invalid={invalid}
          aria-describedby="mimo-key-hint"
          options={[
            { value: "", label: t("mimo.choose") },
            { value: "payg", label: t("mimo.payg") },
            { value: "token_plan", label: t("mimo.plan") },
          ]}
          onValueChange={(billingMode) =>
            onChange({
              ...value,
              billingMode,
              region: billingMode === "token_plan" ? value.region || "cn" : "",
            })
          }
        />
      </div>
      {value.billingMode === "token_plan" ? (
        <div className="space-y-1.5">
          <label htmlFor="mimo-region" className="text-xs">
            {t("mimo.region")}
          </label>
          <SelectField
            id="mimo-region"
            value={value.region}
            disabled={disabled}
            aria-invalid={invalid}
            aria-describedby="mimo-key-hint"
            options={MIMO_PLAN_REGIONS.map((region) => ({
              value: region,
              label: t(`mimo.${region}`),
            }))}
            onValueChange={(region) => onChange({ ...value, region })}
          />
        </div>
      ) : null}
      {validMiMoOptions(value) ? (
        <p className="text-muted-foreground font-mono text-xs break-all">{miMoEndpoint(value)}</p>
      ) : null}
      <p id="mimo-key-hint" className="text-muted-foreground text-xs leading-5">
        {t("mimo.keyHint")}
      </p>
      {invalid ? (
        <p role="alert" className="text-destructive text-xs">
          {t("mimo.choose")}
        </p>
      ) : null}
    </fieldset>
  )
}

export function MiMoBadge({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const value = account.mimo
  if (!value && account.providerId !== "builtin_mimo") return null
  const label =
    value && validMiMoOptions(value)
      ? value.billingMode === "payg"
        ? t("mimo.payg")
        : `Token Plan · ${t(value.region === "cn" ? "mimo.cn" : value.region === "sgp" ? "mimo.sgp" : "mimo.ams")}`
      : t("mimo.needsConfiguration")
  return (
    <span className="bg-muted text-muted-foreground inline-flex rounded-md px-2 py-0.5 text-[11px]">
      {label}
    </span>
  )
}
