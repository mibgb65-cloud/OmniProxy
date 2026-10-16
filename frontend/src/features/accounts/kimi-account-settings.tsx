import { useTranslation } from "react-i18next"
import { AccountSettingsPopover } from "./account-settings-popover"
import { BillingOrderSettings } from "./billing-order-settings"
import { KimiRegionSettings } from "./kimi-region-settings"

export function KimiAccountSettings() {
  const { t } = useTranslation()
  return (
    <AccountSettingsPopover title={t("kimi.settings.title")} closeLabel={t("kimi.settings.close")}>
      <BillingOrderSettings provider="kimi" />
      <KimiRegionSettings />
    </AccountSettingsPopover>
  )
}
