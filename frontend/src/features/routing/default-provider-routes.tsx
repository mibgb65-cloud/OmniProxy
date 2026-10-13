import { useTranslation } from "react-i18next"
import type { DefaultModelRoute } from "../../services/routing"
import { DefaultModelRouteRow } from "./default-model-route-row"
import { groupDefaultRoutesByProvider } from "./routing-groups"

export function DefaultProviderRoutes({
  items,
  search,
  disabled,
  onSelect,
  onConfigurePriority,
}: {
  items: DefaultModelRoute[]
  search: string
  disabled: boolean
  onSelect: (model: string, providerId: string) => void
  onConfigurePriority: (item: DefaultModelRoute) => void
}) {
  const { t } = useTranslation()
  const groups = groupDefaultRoutesByProvider(items, search)
  return (
    <div className="space-y-4">
      {groups.map((group) => {
        const name = group.name || t("routing.grouping.unavailable")
        return (
          <section key={group.id} aria-label={name} className="min-w-0 rounded-lg border">
            <div className="bg-muted/40 flex flex-wrap items-center justify-between gap-2 border-b px-4 py-3">
              <h3 className="text-sm font-semibold wrap-anywhere">{name}</h3>
              <span className="text-muted-foreground text-xs">
                {t("routing.defaults.count", { count: group.items.length })}
              </span>
            </div>
            <ul className="divide-border min-w-0 divide-y" aria-label={name}>
              {group.items.map((item) => (
                <DefaultModelRouteRow
                  key={item.model}
                  item={item}
                  disabled={disabled}
                  groupProviderName={name}
                  onSelect={(providerId) => onSelect(item.model, providerId)}
                  onConfigurePriority={() => onConfigurePriority(item)}
                />
              ))}
            </ul>
          </section>
        )
      })}
      {!groups.length ? (
        <p className="text-muted-foreground text-sm">{t("routing.defaults.noMatches")}</p>
      ) : null}
    </div>
  )
}
