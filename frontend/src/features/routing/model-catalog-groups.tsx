import type { DiscoveredModel } from "../../services/models"
import type { AccountItem, ProviderConfiguration } from "../../services/omniproxy"
import { ModelCatalogRow } from "./model-catalog-row"
import type { ModelProviderView } from "./model-provider-view-toggle"
import { groupCatalogModels } from "./routing-groups"

export function ModelCatalogGroups({
  models,
  accounts,
  providers,
  view,
}: {
  models: DiscoveredModel[]
  accounts: AccountItem[]
  providers: ProviderConfiguration[]
  view: ModelProviderView
}) {
  const groups = groupCatalogModels(models, providers, view)
  return (
    <div className="space-y-4">
      {groups.map((group) => (
        <section key={group.id} aria-label={group.name} className="min-w-0 rounded-lg border">
          <h3 className="bg-muted/40 border-b px-4 py-3 text-sm font-semibold wrap-anywhere">
            {group.name}
          </h3>
          <div className="divide-border divide-y px-4">
            {group.relations.map((relation) => {
              const name = view === "model" ? relation.providerName : relation.model
              return (
                <section
                  key={`${relation.providerId}:${relation.model}`}
                  aria-label={name}
                  className="min-w-0 py-3"
                >
                  <h4 className="text-sm font-medium wrap-anywhere">{name}</h4>
                  {relation.name && relation.name !== relation.model ? (
                    <p className="text-muted-foreground mt-1 text-xs wrap-anywhere">
                      {relation.name}
                    </p>
                  ) : null}
                  <ul>
                    {relation.entries.map((model) => (
                      <ModelCatalogRow
                        key={model.accountId}
                        model={model}
                        account={accounts.find((account) => account.id === model.accountId)}
                      />
                    ))}
                  </ul>
                </section>
              )
            })}
          </div>
        </section>
      ))}
    </div>
  )
}
