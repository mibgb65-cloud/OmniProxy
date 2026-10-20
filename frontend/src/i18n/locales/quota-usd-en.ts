import type { quotaUsdZh } from "./quota-usd-zh"

export const quotaUsdEn: Record<keyof typeof quotaUsdZh, string> = {
  "subscription.usd.title": "Estimated quota in USD",
  "subscription.usd.fiveHour": "Estimated 5h total",
  "subscription.usd.weekly": "Estimated weekly total",
  "subscription.usd.remaining": "Estimated remaining ≈ {{amount}}",
  "subscription.usd.explanation":
    "Estimated from this window's local token API-equivalent cost ÷ official usage fraction; not an official balance. Unknown cache details receive no cache discount. Excludes long-context, speed and tool surcharges. Requires at least 5% usage. Do not add the two windows.",
  "subscription.usd.pending": "Awaiting estimate",
  "subscription.usd.insufficient": "Below 5% used; awaiting estimate",
  "subscription.usd.noUsage": "No local usage in this window",
  "subscription.usd.missingWindow": "Awaiting window and reset time",
  "subscription.usd.expired": "Window reset; refresh quota",
  "subscription.usd.unknownPrice": "{{count}} requests have unknown prices",
  "subscription.usd.missingUsage": "Incomplete usage; awaiting estimate",
  "subscription.usd.unavailable": "Estimate temporarily unavailable",
  "subscription.usd.resetAdjusted": "Only usage after the early reset",
  "subscription.tokens.title": "Token usage",
  "subscription.tokens.scope": "Retained locally",
  "subscription.tokens.scopeExplanation":
    "Only retained local proxy requests for this account, member and workspace. Excludes direct usage in other clients. Clearing request history reduces these totals.",
  "subscription.tokens.input": "Input",
  "subscription.tokens.output": "Output",
  "subscription.tokens.cached": "Cached",
  "subscription.tokens.cacheIncluded": "Cached tokens are included in input",
  "subscription.tokens.partialCache": "Some requests lack cache details.",
  "subscription.tokens.incomplete": "Some requests have incomplete usage.",
  "subscription.tokens.loading": "Loading local usage",
  "subscription.tokens.unavailable": "Usage temporarily unavailable",
}
