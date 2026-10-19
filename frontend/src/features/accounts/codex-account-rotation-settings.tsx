import { useId } from "react"
import { useIsMutating, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { getCodexSettings, setCodexAccountRotationMode } from "../../services/codex-settings"

export function CodexAccountRotationSettings() {
  const { t } = useTranslation()
  const id = useId()
  const client = useQueryClient()
  const saving = useIsMutating({ mutationKey: ["codex-settings"] }) > 0
  const settings = useQuery({ queryKey: ["codex-settings"], queryFn: getCodexSettings })
  const save = useMutation({
    mutationKey: ["codex-settings"],
    mutationFn: setCodexAccountRotationMode,
    onSuccess: async (data) => {
      await client.cancelQueries({ queryKey: ["codex-settings"] })
      client.setQueryData(["codex-settings"], data)
    },
  })
  return (
    <section className="mt-4 space-y-3">
      <fieldset
        disabled={!settings.isSuccess || settings.isFetching || saving}
        className="space-y-3 rounded-lg border p-3 disabled:opacity-60"
      >
        <legend className="sr-only">{t("subscription.rotation.title")}</legend>
        <h3 aria-hidden="true" className="text-sm font-semibold">
          {t("subscription.rotation.title")}
        </h3>
        <div className="grid grid-cols-2 gap-2">
          {(["queue", "balanced"] as const).map((mode) => (
            <label
              key={mode}
              className="has-checked:border-primary has-checked:bg-primary/5 focus-within:ring-ring/50 relative flex min-h-11 cursor-pointer items-center justify-center rounded-lg border px-2 py-2 text-sm focus-within:ring-2"
            >
              <input
                type="radio"
                name={id}
                value={mode}
                checked={(settings.data?.accountRotationMode ?? "queue") === mode}
                onChange={() => save.mutate(mode)}
                className="sr-only"
              />
              {t(`subscription.rotation.${mode}`)}
            </label>
          ))}
        </div>
        <p className="text-muted-foreground text-xs leading-5">
          {t(
            settings.data?.accountRotationMode === "balanced"
              ? "subscription.rotation.balancedDescription"
              : "subscription.rotation.queueDescription",
          )}
        </p>
      </fieldset>
      {save.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("subscription.settings.saveFailed")}
        </p>
      ) : null}
      {save.isPending ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("subscription.settings.saving")}
        </p>
      ) : null}
    </section>
  )
}
