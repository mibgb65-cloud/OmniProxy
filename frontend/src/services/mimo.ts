import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
export type { Options as MiMoOptions } from "../../bindings/omniproxy/internal/provider/mimo"
import type { Options } from "../../bindings/omniproxy/internal/provider/mimo"

export const MIMO_PLAN_REGIONS = ["cn", "sgp", "ams"] as const

export function validMiMoOptions(value: Options) {
  return (
    Number.isSafeInteger(value.version) &&
    value.version >= 0 &&
    ((value.billingMode === "payg" && value.region === "") ||
      (value.billingMode === "token_plan" &&
        MIMO_PLAN_REGIONS.some((region) => region === value.region)))
  )
}

export function validMiMoKey(value: Options, key: string) {
  const plan = /^(tp-|ttp-).+/.test(key.trim())
  return value.billingMode === "token_plan" ? plan : !/^(tp-|ttp-)/.test(key.trim())
}

export function miMoEndpoint(value: Options) {
  return value.billingMode === "token_plan" &&
    MIMO_PLAN_REGIONS.some((region) => region === value.region)
    ? `https://token-plan-${value.region}.xiaomimimo.com/v1`
    : "https://api.xiaomimimo.com/v1"
}

export function miMoAnthropicBaseURL(value: Options) {
  return miMoEndpoint(value).replace(/\/v1$/, "/anthropic")
}

export function openMiMoConsole(accountId: string) {
  return AppService.OpenMiMoConsole(accountId)
}
