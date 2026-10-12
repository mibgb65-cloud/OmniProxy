import { useTranslation } from "react-i18next"
import { PowerButton } from "../../components/common/power-button"

export function AccountPowerButton({
  name,
  enabled,
  disabled,
  onChange,
}: {
  name: string
  enabled: boolean
  disabled: boolean
  onChange: (enabled: boolean) => void
}) {
  const { t } = useTranslation()
  return (
    <PowerButton
      ariaLabel={t("accounts.toggleLabel", { name })}
      actionLabel={t(enabled ? "accounts.power.disable" : "accounts.power.enable")}
      enabled={enabled}
      disabled={disabled}
      onChange={onChange}
    />
  )
}
