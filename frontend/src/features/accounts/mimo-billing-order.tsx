import { useId } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { SelectField } from "../../components/ui/select-field"
import { useMiMoSettings } from "./use-mimo-settings"

export function MiMoBillingOrder() {
  const { t } = useTranslation()
  const id = useId()
  const { settings, save, saving } = useMiMoSettings()
  return (
    <section className="mt-4 space-y-3">
      <div className="space-y-3 rounded-lg border p-3">
        <label htmlFor={id} className="text-sm font-semibold">
          {t("mimo.settings.order")}
        </label>
        <SelectField
          id={id}
          aria-label={t("mimo.settings.order")}
          aria-describedby={`${id}-hint`}
          value={settings.data?.preferredBillingMode ?? ""}
          placeholder={t("mimo.settings.loading")}
          disabled={!settings.isSuccess || settings.isFetching || saving}
          options={[
            { value: "token_plan", label: t("mimo.settings.planFirst") },
            { value: "payg", label: t("mimo.settings.apiFirst") },
          ]}
          onValueChange={(mode) => save.mutate(mode)}
        />
        <p id={`${id}-hint`} className="text-muted-foreground text-sm leading-6">
          {t("mimo.settings.orderDescription")}
        </p>
        <p className="text-muted-foreground text-xs leading-5">{t("mimo.settings.orderScope")}</p>
      </div>
      {settings.isPending ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("mimo.settings.loading")}
        </p>
      ) : null}
      {settings.isError ? (
        <div role="alert" className="space-y-2">
          <p className="text-destructive text-sm">{t("mimo.settings.loadFailed")}</p>
          <Button
            variant="outline"
            disabled={settings.isFetching}
            onClick={() => void settings.refetch()}
          >
            {t("mimo.settings.retry")}
          </Button>
        </div>
      ) : null}
      {save.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("mimo.settings.saveFailed")}
        </p>
      ) : null}
      {saving || save.isSuccess ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t(saving ? "mimo.settings.saving" : "mimo.settings.saved")}
        </p>
      ) : null}
    </section>
  )
}
