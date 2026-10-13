import type { DefaultModelRoute } from "../../services/routing"

export function savedProviderOrder(item: DefaultModelRoute) {
  if (item.providerOrder?.length) return item.providerOrder
  return item.preferredProviderId
    ? [
        {
          id: item.preferredProviderId,
          name: item.preferredProviderName || item.preferredProviderId,
          available: (item.providers ?? []).some(
            (provider) => provider.id === item.preferredProviderId,
          ),
        },
      ]
    : []
}

export function providerPriorityChoices(item: DefaultModelRoute) {
  const saved = savedProviderOrder(item)
  return [
    ...saved,
    ...(item.providers ?? [])
      .filter((provider) => !saved.some((entry) => entry.id === provider.id))
      .map((provider) => ({ ...provider, available: true })),
  ]
}

export function moveProviderPriority<T>(items: T[], index: number, direction: -1 | 1): T[] {
  const target = index + direction
  if (index < 0 || index >= items.length || target < 0 || target >= items.length) return items
  const result = [...items]
  ;[result[index], result[target]] = [result[target], result[index]]
  return result
}
