import { useTranslation } from "react-i18next"
import { AccountSettingsPopover } from "./account-settings-popover"
import { CodexWebSocketSettings } from "./codex-websocket-settings"
import { CodexQuotaActivationSettings } from "./codex-quota-activation-settings"

export function OpenAIAccountSettings() {
  const { t } = useTranslation()
  return (
    <AccountSettingsPopover
      title={t("subscription.settings.title")}
      closeLabel={t("subscription.settings.close")}
    >
      <CodexWebSocketSettings />
      <CodexQuotaActivationSettings />
    </AccountSettingsPopover>
  )
}
