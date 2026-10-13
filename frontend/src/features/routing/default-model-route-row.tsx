import { useTranslation } from "react-i18next"
import { SelectField } from "../../components/ui/select-field"
import { Button } from "../../components/ui/button"
import { ListOrdered } from "lucide-react"
import type { DefaultModelRoute } from "../../services/routing"
import { routingErrorKey } from "./routing-errors"
import { savedProviderOrder, providerPriorityChoices } from "./provider-priority"

export function DefaultModelRouteRow({
  item,
  disabled,
  onSelect,
  groupProviderName,
  onConfigurePriority,
}: {
  item: DefaultModelRoute
  disabled: boolean
  onSelect: (providerId: string) => void
  groupProviderName?: string
  onConfigurePriority: () => void
}) {
  const { t } = useTranslation()
  const providers = item.providers ?? []
  const missingPreference =
    !!item.preferredProviderId && !providers.some((p) => p.id === item.preferredProviderId)
  const result = item.preview
  const order = savedProviderOrder(item)
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
        {!groupProviderName && providers.length ? (
          <ul
            className="flex flex-wrap gap-1.5"
            aria-label={t("routing.grouping.providersFor", { model: item.model })}
          >
            {providers.map((provider) => (
              <li key={provider.id} className="bg-muted rounded-md px-2 py-1 text-xs wrap-anywhere">
                {provider.name}
              </li>
            ))}
          </ul>
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
          aria-label={t(
            groupProviderName ? "routing.grouping.preferLabel" : "routing.defaults.preferLabel",
            { model: item.model, provider: groupProviderName },
          )}
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
        {providerPriorityChoices(item).length > 1 || order.length ? (
          <Button
            variant="outline"
            className="w-full"
            disabled={disabled || result.matched}
            onClick={onConfigurePriority}
            aria-label={t("routing.priority.configureLabel", {
              model: item.model,
              provider: groupProviderName || "",
            })}
          >
            <ListOrdered aria-hidden="true" />
            {t("routing.priority.configure")}
          </Button>
        ) : null}
        {order.length ? (
          <ol
            className="space-y-1 text-xs"
            aria-label={t("routing.priority.savedOrder", { model: item.model })}
          >
            {order.map((entry, index) => (
              <li key={entry.id} className="text-muted-foreground wrap-anywhere">
                {index + 1}. {entry.name}
                {!entry.available ? ` · ${t("routing.priority.unavailable")}` : ""}
              </li>
            ))}
          </ol>
        ) : null}
        <p className="text-muted-foreground text-xs">
          {t(result.matched ? "routing.defaults.ruleHint" : "routing.defaults.providerCount", {
            count: providers.length,
          })}
        </p>
      </div>
    </li>
  )
}
