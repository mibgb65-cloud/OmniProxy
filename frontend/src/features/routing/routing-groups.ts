import type { DiscoveredModel } from "../../services/models"
import type { ProviderConfiguration } from "../../services/omniproxy"
import type { DefaultModelRoute } from "../../services/routing"
import type { ModelProviderView } from "./model-provider-view-toggle"

export function groupDefaultRoutesByProvider(items: DefaultModelRoute[], search: string) {
  const groups = new Map<string, { id: string; name: string; items: DefaultModelRoute[] }>()
  for (const item of items) {
    // 没有可用服务商的模型仍须展示错误与失效偏好，不能在分组时丢失。
    const providers = item.providers?.length ? item.providers : [{ id: "", name: "" }]
    for (const provider of providers) {
      const group = groups.get(provider.id) ?? { ...provider, items: [] }
      if (!group.items.some((existing) => existing.model === item.model)) group.items.push(item)
      groups.set(provider.id, group)
    }
  }
  const query = search.toLowerCase()
  return [...groups.values()]
    .map((group) => ({
      ...group,
      items: group.items.filter((item) =>
        [group.name, item.model, item.name].some((value) => value.toLowerCase().includes(query)),
      ),
    }))
    .filter((group) => group.items.length)
}

export interface CatalogRelation {
  model: string
  name: string
  providerId: string
  providerName: string
  entries: DiscoveredModel[]
}

export function groupCatalogModels(
  models: DiscoveredModel[],
  providers: ProviderConfiguration[],
  view: ModelProviderView,
) {
  const names = new Map(providers.map((provider) => [provider.id, provider.name]))
  const groups = new Map<string, { id: string; name: string; relations: CatalogRelation[] }>()
  for (const model of models) {
    const providerName = names.get(model.providerId) || model.providerId
    const id = view === "model" ? model.model : model.providerId
    const group = groups.get(id) ?? {
      id,
      name: view === "model" ? model.model : providerName,
      relations: [],
    }
    let relation = group.relations.find(
      (entry) => entry.model === model.model && entry.providerId === model.providerId,
    )
    if (!relation) {
      relation = {
        model: model.model,
        name: model.name,
        providerId: model.providerId,
        providerName,
        entries: [],
      }
      group.relations.push(relation)
    }
    relation.entries.push(model)
    groups.set(id, group)
  }
  return [...groups.values()]
}
