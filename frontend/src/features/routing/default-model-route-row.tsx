import { useTranslation } from "react-i18next"
import { SelectField } from "../../components/ui/select-field"
import type { DefaultModelRoute } from "../../services/routing"
import { routingErrorKey } from "./routing-errors"

export function DefaultModelRouteRow({
  item,
  disabled,
  onSelect,
}: {
  item: DefaultModelRoute
  disabled: boolean
  onSelect: (providerId: string) => void
}) {
  const { t } = useTranslation()
  const providers = item.providers ?? []
  const missingPreference =
    !!item.preferredProviderId && !providers.some((p) => p.id === item.preferredProviderId)
  const result = item.preview
  return (
    <li className="grid min-w-0 gap-3 p-4 sm:grid-cols-[minmax(0,1fr)_minmax(12rem,18rem)]">
      <div className="min-w-0 space-y-1.5">
        <div className="flex flex-wrap items-center gap-2">
          <code className="text-sm font-medium wrap-anywhere">{item.model}</code>
          <span className="bg-secondary text-secondary-foreground shrink-0 rounded-md px-2 py-0.5 text-xs">
            {t(result.matched ? "routing.defaults.overridden" : "routing.defaults.automatic")}
          </span>
        </div>
        {item.name !== item.model ? (
          <p className="text-muted-foreground text-xs wrap-anywhere">{item.name}</p>
        ) : null}
        {result.routable ? (
          <p className="text-muted-foreground text-xs wrap-anywhere">
            {t("routing.defaults.target", { provider: result.providerName, model: result.model })}
          </p>
        ) : (
          <p className="text-destructive text-xs">{t(routingErrorKey(result.errorCode))}</p>
        )}
        {missingPreference && !result.matched ? (
          <p className="text-xs text-amber-700 dark:text-amber-400">
            {t(result.routable ? "routing.defaults.fallback" : "routing.defaults.unavailable", {
              provider: result.providerName,
            })}
          </p>
        ) : null}
      </div>
      <div className="min-w-0 space-y-1.5">
        <SelectField
          aria-label={t("routing.defaults.preferLabel", { model: item.model })}
          value={item.preferredProviderId}
          disabled={disabled || result.matched || (!providers.length && !item.preferredProviderId)}
          onValueChange={onSelect}
          options={[
            { value: "", label: t("routing.defaults.autoSelect") },
            ...providers.map((p) => ({ value: p.id, label: p.name })),
            ...(missingPreference
              ? [
                  {
                    value: item.preferredProviderId,
                    label: t("routing.defaults.savedUnavailable", {
                      name: item.preferredProviderName || t("routing.error.provider_missing"),
                    }),
                    disabled: true,
                  },
                ]
              : []),
          ]}
        />
        <p className="text-muted-foreground text-xs">
          {t(result.matched ? "routing.defaults.ruleHint" : "routing.defaults.providerCount", {
            count: providers.length,
          })}
        </p>
      </div>
    </li>
  )
}
