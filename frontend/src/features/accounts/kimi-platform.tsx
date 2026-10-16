import { useTranslation } from "react-i18next"
import { SelectField } from "../../components/ui/select-field"
import type { KimiOptions } from "../../services/kimi"
import type { AccountItem } from "../../services/omniproxy"

export function KimiPlatformFields({
  value,
  onChange,
  disabled,
  invalid = false,
}: {
  value: KimiOptions
  onChange: (value: KimiOptions) => void
  disabled: boolean
  invalid?: boolean
}) {
  const { t } = useTranslation()
  return (
    <fieldset className="space-y-2" disabled={disabled} aria-describedby="kimi-platform-hint">
      <legend className="mb-2 text-sm font-medium">{t("kimi.platform")}</legend>
      <SelectField
        id="kimi-billing"
        aria-label={t("kimi.billing")}
        value={value.billingMode || "payg"}
        disabled={disabled}
        options={[
          { value: "payg", label: t("kimi.api") },
          { value: "coding", label: t("kimi.coding") },
        ]}
        onValueChange={(billingMode) => onChange({ ...value, billingMode })}
      />
      <SelectField
        id="kimi-platform"
        aria-label={t("kimi.platform")}
        value={value.region}
        disabled={disabled}
        aria-invalid={invalid}
        options={[
          { value: "", label: t("kimi.choose") },
          { value: "cn", label: t("kimi.cn") },
          { value: "global", label: t("kimi.global") },
        ]}
        onValueChange={(region) => onChange({ ...value, region })}
      />
      <p id="kimi-platform-hint" className="text-muted-foreground text-xs leading-5">
        {t("kimi.keyHint")}
      </p>
      {invalid ? (
        <p role="alert" className="text-destructive text-xs">
          {t("kimi.choose")}
        </p>
      ) : null}
    </fieldset>
  )
}

export function KimiPlatformBadge({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  if (!account.kimi && account.providerId !== "builtin_kimi") return null
  return (
    <span className="bg-muted text-muted-foreground inline-flex rounded-md px-2 py-0.5 text-[11px]">
      {account.kimi?.billingMode === "coding"
        ? `${t("kimi.coding")} · ${t(account.kimi.region === "cn" ? "kimi.cn" : "kimi.global")}`
        : account.kimi
          ? t(account.kimi.region === "cn" ? "kimi.cn" : "kimi.global")
          : t("kimi.needsConfiguration")}
    </span>
  )
}
