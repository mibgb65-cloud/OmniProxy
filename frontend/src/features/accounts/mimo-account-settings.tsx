import { useTranslation } from "react-i18next"
import { AccountSettingsPopover } from "./account-settings-popover"

export function MiMoAccountSettings() {
  const { t } = useTranslation()
  return (
    <AccountSettingsPopover title={t("mimo.settings.title")} closeLabel={t("mimo.settings.close")}>
      <p className="text-muted-foreground py-6 text-center text-sm">{t("mimo.settings.empty")}</p>
    </AccountSettingsPopover>
  )
}
