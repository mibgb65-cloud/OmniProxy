import { useId } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { AccountSettingsPopover } from "./account-settings-popover"
import { getRiverSettings, setRiverAccountRotationMode } from "../../services/river-settings"

export function RiverAccountSettings() {
  const { t } = useTranslation()
  const id = useId()
  const client = useQueryClient()
  const settings = useQuery({
    queryKey: ["river-settings"],
    queryFn: getRiverSettings,
    retry: false,
  })
  const save = useMutation({
    mutationKey: ["river-settings"],
    mutationFn: setRiverAccountRotationMode,
    onSuccess: async (data) => {
      await client.cancelQueries({ queryKey: ["river-settings"] })
      client.setQueryData(["river-settings"], data)
    },
  })
  return (
    <AccountSettingsPopover
      title={t("river.settings.title")}
      closeLabel={t("river.settings.close")}
    >
      <fieldset
        disabled={!settings.isSuccess || settings.isFetching || save.isPending}
        className="mt-4 space-y-3 rounded-lg border p-3 disabled:opacity-60"
      >
        <legend className="sr-only">{t("subscription.rotation.title")}</legend>
        <h3 aria-hidden="true" className="text-sm font-semibold">
          {t("subscription.rotation.title")}
        </h3>
        <div className="grid grid-cols-2 gap-2">
          {(["queue", "balanced"] as const).map((mode) => (
            <label
              key={mode}
              className="has-checked:border-primary has-checked:bg-primary/5 focus-within:ring-ring/50 flex min-h-11 cursor-pointer items-center justify-center rounded-lg border p-2 text-sm focus-within:ring-2"
            >
              <input
                type="radio"
                name={id}
                value={mode}
                checked={settings.data?.accountRotationMode === mode}
                onChange={() => save.mutate(mode)}
                className="sr-only"
              />
              {t(`river.rotation.${mode}`)}
            </label>
          ))}
        </div>
        <p className="text-muted-foreground text-xs leading-5">
          {t(
            settings.data?.accountRotationMode === "balanced"
              ? "river.rotation.balancedDescription"
              : "river.rotation.queueDescription",
          )}
        </p>
        <p className="text-muted-foreground text-xs leading-5">{t("river.rotation.failureHint")}</p>
      </fieldset>
      {settings.isError || save.isError ? (
        <p role="alert" className="text-destructive mt-3 text-sm">
          {t(settings.isError ? "river.settings.loadFailed" : "subscription.settings.saveFailed")}
        </p>
      ) : null}
      {save.isPending ? (
        <p role="status" className="mt-3 text-sm">
          {t("subscription.settings.saving")}
        </p>
      ) : null}
    </AccountSettingsPopover>
  )
}
