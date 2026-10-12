import { Controller, useFormContext, useWatch } from "react-hook-form"
import { useTranslation } from "react-i18next"
import { SelectField } from "../../components/ui/select-field"
import { Input } from "../../components/ui/input"
import type { RouteInput } from "../../services/routing"
import type { AccountItem, ProviderConfiguration } from "../../services/omniproxy"
import type { DiscoveredModel } from "../../services/models"
export function RoutingEditorFields({
  providers,
  accounts,
  models,
  catalog = [],
}: {
  providers: ProviderConfiguration[]
  accounts: AccountItem[]
  models: string[]
  catalog?: DiscoveredModel[]
}) {
  const { t } = useTranslation()
  const {
    register,
    control,
    setValue,
    formState: { errors, isSubmitting },
  } = useFormContext<RouteInput>()
  const providerId = useWatch({ control, name: "providerId" })
  const alias = useWatch({ control, name: "alias" })
  const model = useWatch({ control, name: "model" })
  const accountId = useWatch({ control, name: "accountId" })
  const availableAccounts = accounts.filter((item) => item.providerId === providerId)
  const provider = providers.find((p) => p.id === providerId)
  const availableModels = [
    ...new Set([
      ...catalog
        .filter((m) => m.providerId === providerId && (!accountId || m.accountId === accountId))
        .map((m) => m.model),
      ...(provider?.kind === "deepseek"
        ? models.filter((m) => m === "deepseek-flash" || m === "deepseek-v4-pro")
        : []),
    ]),
  ]
  const providerOptions = [
    { value: "", label: t("routing.providerChoose") },
    ...(providerId && !providers.some((p) => p.id === providerId)
      ? [{ value: providerId, label: t("routing.error.provider_missing"), disabled: true }]
      : []),
    ...providers.map((p) => {
      const unsupported =
        p.kind !== "deepseek" &&
        !(p.kind === "openai" && catalog.some((m) => m.providerId === p.id))
      return {
        value: p.id,
        disabled: unsupported,
        label:
          p.name +
          (unsupported
            ? ` · ${t("routing.adapterPending")}`
            : !p.enabled
              ? ` · ${t("routing.disabledSuffix")}`
              : ""),
      }
    }),
  ]
  const modelOptions = [
    { value: "", label: t("routing.modelChoose") },
    ...(model && !availableModels.includes(model)
      ? [{ value: model, label: t("routing.error.unsupported_model"), disabled: true }]
      : []),
    ...availableModels.map((value) => ({ value, label: value })),
  ]
  const accountOptions = [
    { value: "", label: t("routing.pool") },
    ...(accountId && !availableAccounts.some((a) => a.id === accountId)
      ? [{ value: accountId, label: t("routing.error.account_missing"), disabled: true }]
      : []),
    ...availableAccounts.map((a) => ({
      value: a.id,
      label: a.name + (!a.enabled ? ` · ${t("routing.disabledSuffix")}` : ""),
    })),
  ]
  return (
    <>
      <div className="space-y-1.5">
        <label htmlFor="route-alias" className="text-sm font-medium">
          {t("routing.alias")}
        </label>
        <Input
          id="route-alias"
          placeholder="coding-fast"
          maxLength={128}
          disabled={isSubmitting}
          {...register("alias")}
          aria-invalid={!!errors.alias}
          aria-describedby="route-alias-hint route-alias-error"
        />
        <p id="route-alias-hint" className="text-muted-foreground text-xs leading-5">
          {t("routing.aliasHint")}
        </p>
        {errors.alias ? (
          <p id="route-alias-error" className="text-destructive text-xs">
            {t(errors.alias.type === "server" ? "routing.aliasDuplicate" : "routing.aliasInvalid")}
          </p>
        ) : null}
      </div>
      <div className="space-y-1.5">
        <label htmlFor="route-provider" className="text-sm font-medium">
          {t("routing.provider")}
        </label>
        <Controller
          control={control}
          name="providerId"
          render={({ field: { ref, name, onBlur, onChange } }) => (
            <SelectField
              id="route-provider"
              aria-label={t("routing.provider")}
              ref={ref}
              name={name}
              value={providerId}
              onBlur={onBlur}
              disabled={isSubmitting}
              aria-invalid={!!errors.providerId}
              options={providerOptions}
              onValueChange={(value) => {
                onChange(value)
                setValue("accountId", "")
                setValue("model", "")
              }}
            />
          )}
        />
      </div>
      <div className="space-y-1.5">
        <label htmlFor="route-model" className="text-sm font-medium">
          {t("routing.model")}
        </label>
        <Controller
          control={control}
          name="model"
          render={({ field: { ref, name, onBlur, onChange } }) => (
            <SelectField
              id="route-model"
              aria-label={t("routing.model")}
              ref={ref}
              name={name}
              value={model}
              onBlur={onBlur}
              disabled={isSubmitting || !providerId}
              aria-invalid={!!errors.model}
              options={modelOptions}
              onValueChange={onChange}
            />
          )}
        />
        {errors.providerId || errors.model ? (
          <p className="text-destructive text-xs">{t("routing.targetInvalid")}</p>
        ) : null}
      </div>
      <div className="space-y-1.5">
        <label htmlFor="route-account" className="text-sm font-medium">
          {t("routing.account")}
        </label>
        <Controller
          control={control}
          name="accountId"
          render={({ field: { ref, name, onBlur, onChange } }) => (
            <SelectField
              id="route-account"
              aria-label={t("routing.account")}
              ref={ref}
              name={name}
              value={accountId}
              onBlur={onBlur}
              disabled={isSubmitting || !providerId}
              aria-invalid={!!errors.accountId}
              options={accountOptions}
              onValueChange={onChange}
            />
          )}
        />
        <p className="text-muted-foreground text-xs leading-5">{t("routing.accountHint")}</p>
        {errors.accountId ? (
          <p className="text-destructive text-xs">{t("routing.accountInvalid")}</p>
        ) : null}
      </div>
      <label className="flex cursor-pointer items-center gap-2 text-sm">
        <input
          type="checkbox"
          className="size-4"
          disabled={isSubmitting}
          {...register("enabled")}
        />
        {t("routing.enabled")}
      </label>
      {models.includes(alias) ? (
        <p className="text-muted-foreground rounded-md border p-3 text-xs leading-5">
          {t("routing.overrideHint")}
        </p>
      ) : null}
    </>
  )
}
