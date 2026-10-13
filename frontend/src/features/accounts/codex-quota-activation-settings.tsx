import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import {
  getCodexSettings,
  getCodexQuotaActivationStatus,
  setCodexQuotaAutoActivation,
} from "../../services/codex-settings"
import { ProviderSwitch } from "../providers/provider-switch"

export function CodexQuotaActivationSettings() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [draft, setDraft] = useState<boolean | null>(null)
  const settings = useQuery({ queryKey: ["codex-settings"], queryFn: getCodexSettings })
  const save = useMutation({
    mutationFn: setCodexQuotaAutoActivation,
    onSuccess: (data) => {
      client.setQueryData(["codex-settings"], data)
      setDraft(null)
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
  const checked = draft ?? settings.data?.autoQuotaActivation ?? false
  const disabled = !settings.isSuccess || settings.isFetching || save.isPending
  return (
    <section className="mt-5 space-y-3">
      <div className="rounded-lg border p-4">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0 space-y-2">
            <h2 className="text-sm font-semibold">{t("subscription.activation.title")}</h2>
            <p className="text-muted-foreground text-sm leading-6">
              {t("subscription.activation.description")}
            </p>
            <p className="text-muted-foreground text-xs leading-5">
              {t("subscription.activation.cost")}
            </p>
          </div>
          <div className="flex min-h-11 shrink-0 items-center">
            <ProviderSwitch
              name="Codex"
              ariaLabel={t("subscription.activation.title")}
              checked={checked}
              disabled={disabled}
              onChange={(enabled) => {
                setDraft(enabled)
                save.reset()
              }}
            />
          </div>
        </div>
        <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
          <p className="text-muted-foreground text-xs">{t("subscription.activation.saveHint")}</p>
          <Button
            variant="outline"
            disabled={disabled || draft === null || checked === settings.data?.autoQuotaActivation}
            onClick={() => save.mutate(checked)}
          >
            {t(save.isPending ? "subscription.settings.saving" : "subscription.activation.save")}
          </Button>
        </div>
      </div>
      {save.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("subscription.activation.saveFailed")}
        </p>
      ) : null}
      {save.isSuccess ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("subscription.activation.saved")}
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
