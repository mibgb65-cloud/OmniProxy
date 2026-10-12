import { useTranslation } from "react-i18next"
import { PowerButton } from "../../components/common/power-button"

export function ProviderPowerButton({
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
      ariaLabel={t("providers.toggleLabel", { name })}
      actionLabel={t(enabled ? "providers.power.disable" : "providers.power.enable")}
      enabled={enabled}
      disabled={disabled}
      onChange={onChange}
    />
  )
}
