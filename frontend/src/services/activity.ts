import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type {
  RequestQuery,
  RequestItem,
  UsageInfo,
  Breakdown,
  ProviderModel,
} from "../../bindings/omniproxy/internal/application/activity"
export type { RequestItem, UsageInfo, Breakdown }
export type ProviderModelUsage = ProviderModel
export const readUsage = () => AppService.UsageRead()
export const readRequests = (query: RequestQuery) => AppService.RequestsRead(query)
export const readRequest = (id: string) => AppService.RequestRead(id)
export const readRetention = () => AppService.RequestRetention()
export const setRetention = (limit: number) => AppService.RequestRetentionSet(limit)

export function formatTokens(value: string, language = "zh") {
  try {
    return new Intl.NumberFormat(language === "zh" ? "zh-CN" : "en-US").format(BigInt(value))
  } catch {
    return "—"
  }
}
export function validateRequestSearch(search: Record<string, unknown>) {
  return {
    requestId:
      typeof search.requestId === "string" && /^[a-f0-9]{32}$/.test(search.requestId)
        ? search.requestId
        : undefined,
  }
}
