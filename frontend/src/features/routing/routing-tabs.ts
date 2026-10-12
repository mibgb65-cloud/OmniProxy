import type { SearchSchemaInput } from "@tanstack/react-router"

export const routingTabs = ["defaults", "rules", "catalog", "preview"] as const

export type RoutingTab = (typeof routingTabs)[number]

export function isRoutingTab(value: unknown): value is RoutingTab {
  return typeof value === "string" && routingTabs.some((tab) => tab === value)
}

export function validateRoutingSearch(search: SearchSchemaInput & { tab?: unknown }): {
  tab: RoutingTab
} {
  // 显式覆盖非法参数，避免 Router 合并原始值后使所有面板隐藏。
  return { tab: isRoutingTab(search.tab) ? search.tab : "defaults" }
}
