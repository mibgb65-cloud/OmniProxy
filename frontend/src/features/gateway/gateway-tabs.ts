import type { SearchSchemaInput } from "@tanstack/react-router"
import { isAgentClientId, type AgentClientId } from "./agent-clients"

export const gatewayTabs = ["overview", "clients", "manual", "diagnostics"] as const

export type GatewayTab = (typeof gatewayTabs)[number]

export function isGatewayTab(value: unknown): value is GatewayTab {
  return typeof value === "string" && gatewayTabs.some((tab) => tab === value)
}

export function validateGatewaySearch(
  search: SearchSchemaInput & { tab?: unknown; client?: unknown },
): {
  tab: GatewayTab
  client?: AgentClientId
} {
  // Router 会合并原始参数；显式覆盖非法值，避免未知值让所有面板都隐藏。
  return {
    tab: isGatewayTab(search.tab) ? search.tab : "overview",
    ...(Object.hasOwn(search, "client")
      ? { client: isAgentClientId(search.client) ? search.client : undefined }
      : {}),
  }
}
