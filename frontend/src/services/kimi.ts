export type {
  Options as KimiOptions,
  Balance as KimiBalance,
} from "../../bindings/omniproxy/internal/provider/kimi"

export function validKimiOptions(value: { region: string; billingMode?: string; version: number }) {
  return (
    ["cn", "global"].includes(value.region) &&
    (value.billingMode === undefined ||
      value.billingMode === "" ||
      ["payg", "coding"].includes(value.billingMode)) &&
    Number.isSafeInteger(value.version) &&
    value.version >= 0
  )
}
