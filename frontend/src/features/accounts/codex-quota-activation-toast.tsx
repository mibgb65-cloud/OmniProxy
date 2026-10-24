import { useTranslation } from "react-i18next"
import { StatusToast } from "../../components/common/status-toast"
import type { CodexActivationStatus } from "../../services/codex-settings"
import { activationReasonKey } from "./activation-reason"

export function CodexQuotaActivationToast({ status }: { status: CodexActivationStatus }) {
  const { t } = useTranslation()
  const failed = status.failed > 0 || !!status.errorCode
  const reason = activationReasonKey(status.errorCode)
  return (
    <StatusToast
      title={t("subscription.activation.summary", {
        checked: status.checked,
        sent: status.sent,
        activated: status.activated,
      })}
      closeLabel={t("subscription.activation.dismissNotice")}
      tone={failed ? "warning" : status.pending > 0 ? "info" : "success"}
      description={
        status.pending > 0 || failed ? (
          <>
            {status.pending > 0 ? <p>{t("subscription.activation.pending")}</p> : null}
            {failed ? <p>{t(reason ?? "subscription.activation.failed")}</p> : null}
          </>
        ) : undefined
      }
    />
  )
}
