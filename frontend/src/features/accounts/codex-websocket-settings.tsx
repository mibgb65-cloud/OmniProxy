import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { getCodexSettings, setCodexWebSocketEnabled } from "../../services/codex-settings"
import { ProviderSwitch } from "../providers/provider-switch"

export function CodexWebSocketSettings() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const settings = useQuery({ queryKey: ["codex-settings"], queryFn: getCodexSettings })
  const save = useMutation({
    mutationFn: setCodexWebSocketEnabled,
    onSuccess: (data) => client.setQueryData(["codex-settings"], data),
  })
  return (
    <section className="mt-5 space-y-4">
      <div className="flex items-start justify-between gap-4 rounded-lg border p-4">
        <div className="min-w-0 space-y-2">
          <h2 className="text-sm font-semibold">{t("subscription.settings.websocket")}</h2>
          <p className="text-muted-foreground text-sm leading-6">
            {t("subscription.settings.websocketDescription")}
          </p>
          <p className="text-muted-foreground text-xs leading-5">
            {t("subscription.settings.websocketScope")}
          </p>
        </div>
        <div className="flex min-h-11 shrink-0 items-center">
          <ProviderSwitch
            name="Codex"
            ariaLabel={t("subscription.settings.websocket")}
            checked={settings.data?.webSocketEnabled ?? false}
            disabled={!settings.isSuccess || settings.isFetching || save.isPending}
            onChange={(enabled) => save.mutate(enabled)}
          />
        </div>
      </div>
      {settings.isPending ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("subscription.settings.loading")}
        </p>
      ) : null}
      {settings.isError ? (
        <div role="alert" className="space-y-2">
          <p className="text-destructive text-sm">{t("subscription.settings.loadFailed")}</p>
          <Button
            variant="outline"
            disabled={settings.isFetching}
            onClick={() => void settings.refetch()}
          >
            {t("subscription.settings.retry")}
          </Button>
        </div>
      ) : null}
      {save.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("subscription.settings.saveFailed")}
        </p>
      ) : null}
      {save.isPending || save.isSuccess ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t(save.isPending ? "subscription.settings.saving" : "subscription.settings.saved")}
        </p>
      ) : null}
    </section>
  )
}
