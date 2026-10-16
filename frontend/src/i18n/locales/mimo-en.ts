export const mimoEn = {
  "mimo.settings.title": "Xiaomi MiMo account settings",
  "mimo.settings.close": "Close account settings",
  "mimo.settings.order": "Default usage order",
  "mimo.settings.planFirst": "Token Plan → API Key (pay-as-you-go)",
  "mimo.settings.apiFirst": "API Key (pay-as-you-go) → Token Plan",
  "mimo.settings.orderDescription":
    "Each model is listed once. Use the preferred channel first, then the next channel if no enabled account supports the model.",
  "mimo.settings.orderScope":
    "Changes save automatically and apply to all Xiaomi MiMo accounts. Fixed accounts and explicit channels take precedence. Accounts in the same channel follow creation order; failed requests are not retried through another channel.",
  "mimo.settings.loading": "Loading default order…",
  "mimo.settings.loadFailed": "Could not load the default usage order. Try again.",
  "mimo.settings.retry": "Reload",
  "mimo.settings.saveFailed": "Save failed. The previous order is unchanged. Try again.",
  "mimo.settings.saving": "Saving…",
  "mimo.settings.saved": "Default usage order saved and applied.",
  "mimo.openAccess": "View Token Plan access details",
  "mimo.closeAccess": "Close Token Plan access details",
  "mimo.accessTitle": "Token Plan access",
  "mimo.accessDescription":
    "These are base URLs for the compatible protocols. Use a personal tp- or team ttp- Key, and select the cluster shown in the console when configuring the account.",
  "mimo.accessModels":
    "The plan supports mimo-v2.6-pro and mimo-v2.6-flash. Ultraspeed uses the pay-as-you-go API.",
  "mimo.apiEndpoint": "Pay-as-you-go",
  "mimo.probePayg": "Test pay-as-you-go",
  "mimo.probeScope":
    "The card tests the default pay-as-you-go endpoint. Token Plan calls use the cluster selected for the account.",
  "mimo.connection": "Xiaomi MiMo connection",
  "mimo.billing": "Billing mode",
  "mimo.payg": "Pay-as-you-go API",
  "mimo.plan": "Token Plan",
  "mimo.region": "Plan cluster",
  "mimo.cn": "China",
  "mimo.sgp": "Singapore",
  "mimo.ams": "Europe",
  "mimo.choose": "Choose a billing mode and plan cluster.",
  "mimo.needsConfiguration": "MiMo channel needs confirmation",
  "mimo.keyHint":
    "Use an ordinary Key for pay-as-you-go, or a personal tp- / team ttp- Key for Token Plan. Select the cluster shown in the console. A batch shares this selection.",
  "mimo.routingHint":
    "Pay-as-you-go and Token Plan use separate billing. Shared model names follow the default usage order in settings; fixed accounts and explicit channels are also supported. Failed requests are not retried through another channel.",
  "mimo.quotaTitle": "Token Plan quota",
  "mimo.quotaUnknown":
    "Official remaining Credits have not been queried. See Usage for local token counts, and the official console for plan allowance and validity.",
  "mimo.openConsole": "View official plan",
  "mimo.openFailed": "Could not open the official console. Try again later.",
  "routing.error.ambiguous_mimo_pool":
    "Multiple Xiaomi MiMo billing channels or clusters are available. Choose a channel model or fixed account.",
} as const
