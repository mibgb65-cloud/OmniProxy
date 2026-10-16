import { useTranslation } from "react-i18next"
import { AccountSettingsPopover } from "./account-settings-popover"
import { BillingOrderSettings } from "./billing-order-settings"

export function MiMoAccountSettings() {
  const { t } = useTranslation()
  return (
    <AccountSettingsPopover title={t("mimo.settings.title")} closeLabel={t("mimo.settings.close")}>
      <BillingOrderSettings provider="mimo" />
    </AccountSettingsPopover>
  )
}
