import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type { Info } from "../../bindings/omniproxy/internal/application/balance"

export type AccountBalanceInfo = Info
export const readAccountBalance = async (id: string) => AppService.AccountBalance(id)

export function balanceError(error: unknown): string {
  const text = error instanceof Error ? error.message : String(error)
  const codes = [
    "unauthorized",
    "forbidden",
    "rate_limited",
    "service_unavailable",
    "network_failed",
    "invalid_response",
    "credential_unavailable",
    "unsupported",
    "configuration_unavailable",
  ]
  return codes.find((code) => text === `balance: ${code}`) || "query_failed"
}
