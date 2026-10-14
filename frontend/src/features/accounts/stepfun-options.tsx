import { useTranslation } from "react-i18next"
import { SelectField } from "../../components/ui/select-field"
import type { StepFunOptions as Options } from "../../services/stepfun"
import type { AccountItem } from "../../services/omniproxy"

export function StepFunOptionsFields({
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
    <fieldset className="space-y-3" disabled={disabled} aria-describedby="stepfun-key-hint">
      <legend className="mb-2 text-sm font-medium">{t("stepfun.connection")}</legend>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <label htmlFor="stepfun-region" className="text-xs">
            {t("stepfun.region")}
          </label>
          <SelectField
            id="stepfun-region"
            value={value.region}
            disabled={disabled}
            aria-invalid={invalid}
            options={[
              { value: "", label: t("stepfun.choose") },
              { value: "cn", label: t("stepfun.cn") },
              { value: "global", label: t("stepfun.global") },
            ]}
            onValueChange={(region) => onChange({ ...value, region })}
          />
        </div>
        <div className="space-y-1.5">
          <label htmlFor="stepfun-billing" className="text-xs">
            {t("stepfun.billing")}
          </label>
          <SelectField
            id="stepfun-billing"
            value={value.billingMode}
            disabled={disabled}
            aria-invalid={invalid}
            options={[
              { value: "", label: t("stepfun.choose") },
              { value: "payg", label: t("stepfun.payg") },
              { value: "step_plan", label: t("stepfun.plan") },
            ]}
            onValueChange={(billingMode) => onChange({ ...value, billingMode })}
          />
        </div>
      </div>
      <p id="stepfun-key-hint" className="text-muted-foreground text-xs leading-5">
        {t("stepfun.keyHint")}
      </p>
      {invalid ? (
        <p role="alert" className="text-destructive text-xs">
          {t("stepfun.choose")}
        </p>
      ) : null}
    </fieldset>
  )
}

export function StepFunBadge({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const value = account.stepfun
  if (!value && account.providerId !== "builtin_stepfun") return null
  return (
    <span className="bg-muted text-muted-foreground inline-flex rounded-md px-2 py-0.5 text-[11px]">
      {value
        ? `${t(value.region === "cn" ? "stepfun.cn" : "stepfun.global")} · ${t(value.billingMode === "payg" ? "stepfun.payg" : "stepfun.plan")}`
        : t("stepfun.needsConfiguration")}
    </span>
  )
}
