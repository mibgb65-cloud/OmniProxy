import { useIsMutating, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import {
  getCodexSettings,
  getCodexQuotaActivationStatus,
  setCodexQuotaAutoActivation,
} from "../../services/codex-settings"
import { ProviderSwitch } from "../providers/provider-switch"

export function CodexQuotaActivationSettings() {
  const { t } = useTranslation()
  const client = useQueryClient()
  // 两个开关返回完整设置，保存期间共享锁，避免响应乱序覆盖另一开关。
  const saving = useIsMutating({ mutationKey: ["codex-settings"] }) > 0
  const settings = useQuery({ queryKey: ["codex-settings"], queryFn: getCodexSettings })
  const save = useMutation({
    mutationKey: ["codex-settings"],
    mutationFn: setCodexQuotaAutoActivation,
    onSuccess: async (data) => {
      await client.cancelQueries({ queryKey: ["codex-settings"] })
      client.setQueryData(["codex-settings"], data)
      void client.invalidateQueries({ queryKey: ["codex-quota-activation"] })
    },
  })
  const status = useQuery({
    queryKey: ["codex-quota-activation"],
    queryFn: getCodexQuotaActivationStatus,
    enabled: settings.data?.autoQuotaActivation === true,
    retry: false,
    refetchInterval: settings.data?.autoQuotaActivation ? 2000 : false,
  })
  const checked = settings.data?.autoQuotaActivation ?? false
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
              onChange={(enabled) => save.mutate(enabled)}
            />
          </div>
        </div>
        <p className="text-muted-foreground text-sm leading-6">
          {t("subscription.activation.description")}
        </p>
        <p className="text-muted-foreground text-xs leading-5">
          {t("subscription.activation.cost")}
        </p>
        <p className="text-muted-foreground text-xs leading-5">
          {t("subscription.activation.saveHint")}
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
      {settings.data?.autoQuotaActivation && status.data ? (
        <div role="status" className="text-muted-foreground space-y-1 text-xs">
          <p>
            {t(
              status.data.running
                ? "subscription.activation.checking"
                : "subscription.activation.summary",
              {
                checked: status.data.checked,
                sent: status.data.sent,
                activated: status.data.activated,
              },
            )}
          </p>
          {status.data.pending > 0 ? <p>{t("subscription.activation.pending")}</p> : null}
          {status.data.failed > 0 || status.data.errorCode ? (
            <p>{t("subscription.activation.failed")}</p>
          ) : null}
        </div>
      ) : null}
      {status.isError && settings.data?.autoQuotaActivation ? (
        <p role="alert" className="text-destructive text-sm">
          {t("subscription.activation.statusFailed")}
        </p>
      ) : null}
    </section>
  )
}
