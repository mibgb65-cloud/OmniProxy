export const kimiSettingsEn = {
  "kimi.settings.title": "Kimi account settings",
  "kimi.settings.close": "Close account settings",
  "kimi.settings.order": "Default usage order",
  "kimi.settings.planFirst": "Kimi Code subscription → API Key (pay-as-you-go)",
  "kimi.settings.apiFirst": "API Key (pay-as-you-go) → Kimi Code subscription",
  "kimi.settings.orderDescription":
    "Shared models are listed once. Select the billing channel first, then an enabled account supporting the model by region. Use the next channel when no preferred account qualifies.",
  "kimi.settings.orderScope":
    "Changes save automatically. Fixed accounts and explicit channels take precedence; failed requests are not retried through another channel. The subscription may use an enabled official booster wallet; its settings are unchanged.",
  "kimi.settings.region": "Default region order",
  "kimi.settings.cnFirst": "China → Overseas",
  "kimi.settings.globalFirst": "Overseas → China",
  "kimi.settings.regionDescription":
    "Prefer this region within the selected billing channel; use the other region when no account qualifies. Accounts in the same region follow creation order.",
  "kimi.settings.loading": "Loading default order…",
  "kimi.settings.loadFailed": "Could not load the default usage order. Try again.",
  "kimi.settings.retry": "Reload",
  "kimi.settings.saveFailed": "Save failed. The previous order is unchanged. Try again.",
  "kimi.settings.saving": "Saving…",
  "kimi.settings.saved": "Default usage order saved and applied.",
  "kimi.billing": "Billing mode",
  "kimi.api": "Pay-as-you-go API",
  "kimi.coding": "Kimi Code subscription",
  "kimi.codeQuota": "Kimi Code subscription quota",
  "kimi.codeUnknown":
    "Official remaining quota has not been queried. Check the console; the ordinary API balance is separate. Model and context permissions depend on the membership plan.",
  "kimi.openCodeConsole": "Open subscription console",
  "kimi.openCodeFailed": "Could not open the subscription console. Try again.",
} as const
