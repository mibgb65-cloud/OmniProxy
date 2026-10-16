import { useId } from "react"
import { useIsMutating, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { SelectField } from "../../components/ui/select-field"
import { getKimiSettings, setKimiPreferredRegion } from "../../services/kimi-settings"

export function KimiRegionSettings() {
  const { t } = useTranslation()
  const id = useId()
  const client = useQueryClient()
  const settings = useQuery({ queryKey: ["kimi-settings"], queryFn: getKimiSettings })
  const saving = useIsMutating({ mutationKey: ["kimi-settings"] }) > 0
  const save = useMutation({
    mutationKey: ["kimi-settings"],
    mutationFn: setKimiPreferredRegion,
    onSuccess: async (data) => {
      await client.cancelQueries({ queryKey: ["kimi-settings"] })
      client.setQueryData(["kimi-settings"], data)
      await Promise.all(
        ["routes", "default-routes", "gateway-runtime", "agent-clients"].map((key) =>
          client.invalidateQueries({ queryKey: [key] }),
        ),
      )
    },
  })
  return (
    <section className="mt-4 space-y-3" aria-label={t("kimi.settings.region")}>
      <div className="space-y-3 rounded-lg border p-3">
        <label htmlFor={id} className="text-sm font-semibold">
          {t("kimi.settings.region")}
        </label>
        <SelectField
          id={id}
          aria-label={t("kimi.settings.region")}
          aria-describedby={`${id}-hint`}
          value={settings.data?.preferredRegion ?? ""}
          disabled={!settings.isSuccess || settings.isFetching || saving}
          placeholder={t("kimi.settings.loading")}
          options={[
            { value: "cn", label: t("kimi.settings.cnFirst") },
            { value: "global", label: t("kimi.settings.globalFirst") },
          ]}
          onValueChange={(region) => save.mutate(region)}
        />
        <p id={`${id}-hint`} className="text-muted-foreground text-xs leading-5">
          {t("kimi.settings.regionDescription")}
        </p>
      </div>
      {save.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("kimi.settings.saveFailed")}
        </p>
      ) : null}
      {save.isSuccess ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("kimi.settings.saved")}
        </p>
      ) : null}
    </section>
  )
}
