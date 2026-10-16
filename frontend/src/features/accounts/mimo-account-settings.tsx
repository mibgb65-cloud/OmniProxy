import { useTranslation } from "react-i18next"
import { AccountSettingsPopover } from "./account-settings-popover"
import { MiMoBillingOrder } from "./mimo-billing-order"

export function MiMoAccountSettings() {
  const { t } = useTranslation()
  return (
    <AccountSettingsPopover title={t("mimo.settings.title")} closeLabel={t("mimo.settings.close")}>
      <MiMoBillingOrder />
    </AccountSettingsPopover>
  )
}
