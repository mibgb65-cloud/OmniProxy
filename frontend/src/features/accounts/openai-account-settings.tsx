import { useTranslation } from "react-i18next"
import { useCallback, useRef, useState } from "react"
import { AccountSettingsPopover } from "./account-settings-popover"
import { CodexWebSocketSettings } from "./codex-websocket-settings"
import { CodexQuotaActivationSettings } from "./codex-quota-activation-settings"
import { CodexAccountRotationSettings } from "./codex-account-rotation-settings"
import { CodexQuotaActivationToast } from "./codex-quota-activation-toast"
import type { CodexActivationStatus } from "../../services/codex-settings"

export function OpenAIAccountSettings() {
  const { t } = useTranslation()
  const seen = useRef("")
  const [notice, setNotice] = useState<{ key: string; status: CodexActivationStatus }>()
  const onResult = useCallback((status: CodexActivationStatus) => {
    const key = JSON.stringify(status)
    if (seen.current === key) return
    seen.current = key
    setNotice({ key, status })
  }, [])
  return (
    <>
      <AccountSettingsPopover
        title={t("subscription.settings.title")}
        closeLabel={t("subscription.settings.close")}
      >
        <CodexWebSocketSettings />
        <CodexQuotaActivationSettings onResult={onResult} />
        <CodexAccountRotationSettings />
      </AccountSettingsPopover>
      {notice ? <CodexQuotaActivationToast key={notice.key} status={notice.status} /> : null}
    </>
  )
}
