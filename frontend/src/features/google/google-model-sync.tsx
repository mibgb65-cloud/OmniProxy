import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { syncAccountModels } from "../../services/models"

export function GoogleModelSync({ accountId, disabled }: { accountId: string; disabled: boolean }) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const mutation = useMutation({
    mutationFn: () => syncAccountModels(accountId),
    onSuccess: async () => {
      await Promise.all(
        ["model-catalog", "default-routes", "routes", "agent-clients"].map((key) =>
          client.invalidateQueries({ queryKey: [key] }),
        ),
      )
    },
  })
  return (
    <div className="space-y-2">
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={disabled || mutation.isPending}
        onClick={() => mutation.mutate()}
      >
        {t(mutation.isPending ? "google.models.syncing" : "google.models.sync")}
      </Button>
      {mutation.isSuccess ? (
        <p role="status" className="text-muted-foreground text-xs">
          {t("google.models.synced")}
        </p>
      ) : null}
      {mutation.isError ? (
        <p role="alert" className="text-destructive text-xs">
          {t("google.models.syncFailed")}
        </p>
      ) : null}
    </div>
  )
}
