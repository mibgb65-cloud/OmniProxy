import type { riverSettingsZh } from "./river-settings-zh"

export const riverSettingsEn: Record<keyof typeof riverSettingsZh, string> = {
  "river.usage.notice":
    "Retained local proxy requests only, not River's official bill or balance. Cached tokens are included in input. “—” means details were not reported; “≥” marks a partial known total.",
  "river.usage.loading": "Reading local usage…",
  "river.usage.failed": "Cannot read local usage.",
  "river.usage.retry": "Retry",
  "river.usage.incomplete":
    "Includes requests in progress or with incomplete usage. Values reflect recorded usage only.",
  "river.settings.title": "River account settings",
  "river.settings.close": "Close River account settings",
  "river.settings.loadFailed": "Cannot read River settings. Close and try again.",
  "river.rotation.queue": "Queue",
  "river.rotation.balanced": "Balanced",
  "river.rotation.queueDescription":
    "Use accounts in order, switching to another eligible key in the same pool when unavailable.",
  "river.rotation.balancedDescription":
    "Distribute new requests across eligible keys in the same pool. Balances request counts, not exact token usage.",
  "river.rotation.failureHint":
    "Uses the gateway failover switch. Temporarily skips a key after two consecutive network or server failures. Uncertain requests are not replayed; fixed accounts never switch.",
}
