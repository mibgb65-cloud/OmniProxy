import { useIsMutating, useMutation, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { setCodexQuotaAutoActivation } from "../../services/codex-settings"
import { ProviderSwitch } from "../providers/provider-switch"
import { activationReasonKey } from "./activation-reason"
import { codexActivationStatusKey, useCodexQuotaActivation } from "./use-codex-quota-activation"

export function CodexQuotaActivationSettings() {
  const { t } = useTranslation()
  const client = useQueryClient()
  // 各项设置返回完整对象，保存期间共享锁，避免响应乱序覆盖另一设置。
  const saving = useIsMutating({ mutationKey: ["codex-settings"] }) > 0
  const { settings, status, enabled: checked } = useCodexQuotaActivation()
  const save = useMutation({
    mutationKey: ["codex-settings"],
    mutationFn: setCodexQuotaAutoActivation,
    onSuccess: async (data) => {
      await client.cancelQueries({ queryKey: ["codex-settings"] })
      client.setQueryData(["codex-settings"], data)
      void client.invalidateQueries({ queryKey: codexActivationStatusKey })
    },
  })
  const disabled = !settings.isSuccess || settings.isFetching || saving
  return (
    <section className="mt-4 space-y-3">
      <div className="space-y-2 rounded-lg border p-3">
        <div className="flex items-center justify-between gap-3">
          <h3 className="text-sm font-semibold">{t("subscription.activation.title")}</h3>
          <div className="flex min-h-11 shrink-0 items-center">
            <ProviderSwitch
              name="Codex"
              ariaLabel={t("subscription.activation.title")}
              checked={checked}
              disabled={disabled}
              onChange={(next) => save.mutate(next)}
            />
          </div>
        </div>
        <p className="text-muted-foreground text-sm leading-6">
          {t("subscription.activation.description")}
        </p>
      </div>
      {save.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("subscription.activation.saveFailed")}
        </p>
      ) : null}
      {save.isPending || save.isSuccess ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t(save.isPending ? "subscription.settings.saving" : "subscription.activation.saved")}
        </p>
      ) : null}
      {checked && status.data?.running ? (
        <p role="status" className="text-muted-foreground text-xs">
          {t("subscription.activation.checking")}
        </p>
      ) : null}
      {checked && !status.data?.running && status.data?.errorCode ? (
        <p role="status" className="text-muted-foreground text-xs">
          {t(activationReasonKey(status.data.errorCode) ?? "subscription.activation.failed")}
        </p>
      ) : null}
      {status.isError && checked ? (
        <p role="alert" className="text-destructive text-sm">
          {t("subscription.activation.statusFailed")}
        </p>
      ) : null}
    </section>
  )
}
