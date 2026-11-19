import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type { RequestExportResult } from "../../bindings/omniproxy/internal/transport/wails"
import type {
  RequestQuery,
  RequestItem,
  RequestContent,
  UsageInfo,
  Breakdown,
  DayUsage,
  ProviderModel,
  Summary,
  HourUsage,
  ModelDay,
  CacheUsage,
  Reliability,
} from "../../bindings/omniproxy/internal/application/activity"
export type {
  RequestItem,
  RequestContent,
  RequestExportResult,
  UsageInfo,
  Breakdown,
  DayUsage,
  Summary,
  HourUsage,
  ModelDay,
  CacheUsage,
  Reliability,
}
export type ProviderModelUsage = ProviderModel
export const readUsage = () => AppService.UsageRead()
export const readAccountTokenUsage = (id: string) => AppService.AccountTokenUsage(id)
export const readRequests = (query: RequestQuery) => AppService.RequestsRead(query)
export const readRequest = (id: string) => AppService.RequestRead(id)
export const readRequestContent = (id: string) => AppService.RequestContentRead(id)
export const readRetention = () => AppService.RequestRetention()
export const setRetention = (limit: number) => AppService.RequestRetentionSet(limit)
export const setRequestContentRecording = (enabled: boolean) =>
  AppService.RequestContentSet(enabled)
export const clearRequestContent = () => AppService.RequestContentClear()
export const exportRequests = (query: RequestQuery, language: string) =>
  AppService.RequestsExport(query, language)

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
