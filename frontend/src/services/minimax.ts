import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type { Options } from "../../bindings/omniproxy/internal/provider/minimax"
export type {
  Options as MiniMaxOptions,
  Quota as MiniMaxQuota,
  QuotaWindow as MiniMaxQuotaWindow,
} from "../../bindings/omniproxy/internal/provider/minimax"

export function validMiniMaxOptions(value?: Options | null) {
  return (
    !!value &&
    ["cn", "global"].includes(value.region) &&
    ["payg", "subscription"].includes(value.billingMode)
  )
}
export const readMiniMaxQuota = (id: string) => AppService.MiniMaxQuota(id)
export const openMiniMaxConsole = (id: string) => AppService.OpenMiniMaxConsole(id)

/** 仅显示稳定分类，避免把原始上游响应带入界面。 */
export function miniMaxError(error: unknown) {
  const text = error instanceof Error ? error.message : String(error)
  const known = [
    "unauthorized",
    "forbidden",
    "rate_limited",
    "network_failed",
    "invalid_response",
    "configuration_unavailable",
    "credential_unavailable",
    "invalid_configuration",
  ] as const
  return known.find((code) => text.includes(`minimax: ${code}`)) ?? "query_failed"
}
