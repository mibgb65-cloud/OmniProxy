import { useTranslation } from "react-i18next"
import { AccountSettingsPopover } from "./account-settings-popover"
import { BillingOrderSettings } from "./billing-order-settings"

export function StepFunAccountSettings() {
  const { t } = useTranslation()
  return (
    <AccountSettingsPopover
      title={t("stepfun.settings.title")}
      closeLabel={t("stepfun.settings.close")}
    >
      <BillingOrderSettings provider="stepfun" />
    </AccountSettingsPopover>
  )
}
