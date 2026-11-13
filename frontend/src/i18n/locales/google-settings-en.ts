export const googleSettingsEn = {
  "google.settings.title": "Google account settings",
  "google.settings.close": "Close Google account settings",
  "google.settings.loadFailed": "Unable to read Google settings. Close this panel and retry.",
  "google.rotation.queueDescription":
    "Uses your Google AI subscription accounts in order and moves to the next available account in the same pool when quotas run out or an account is unavailable.",
  "google.rotation.balancedDescription":
    "Spreads new requests across the available Google subscription accounts in the same pool. It balances request counts, not token consumption.",
  "google.rotation.failureHint":
    "Applies to Google AI subscription accounts only; API key accounts never rotate. Follows the gateway failover switch: an account is skipped after two consecutive network or server failures, requests with an uncertain outcome are never replayed, and a pinned account does not switch.",
} as const
