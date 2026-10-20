import type { quotaUsdZh } from "./quota-usd-zh"

export const quotaUsdEn: Record<keyof typeof quotaUsdZh, string> = {
  "subscription.usd.title": "Estimated quota in USD",
  "subscription.usd.fiveHour": "Estimated 5h total",
  "subscription.usd.weekly": "Estimated weekly total",
  "subscription.usd.remaining": "Estimated remaining ≈ {{amount}}",
  "subscription.usd.scope": "Local sample reference, not an official balance",
  "subscription.usd.explanation":
    "Uses the previous version's formula: base API-equivalent cost of retained local proxy tokens in this window ÷ official usage fraction. Requires at least 5% usage. Each actual model uses its reference price, without counting cache reads twice. Missing historical cache details produce a range. Prices are not live and exclude long-context, speed and tool surcharges. Other clients, cleared history and price changes can underestimate or distort the result. Do not add the windows together; this is not a cash balance.",
  "subscription.usd.pending": "Awaiting estimate",
  "subscription.usd.insufficient": "Below 5% used; awaiting estimate",
  "subscription.usd.noUsage": "No local usage in this window",
  "subscription.usd.missingWindow": "Awaiting window and reset time",
  "subscription.usd.expired": "Window reset; refresh quota",
  "subscription.usd.unknownPrice": "{{count}} requests have unknown prices",
  "subscription.usd.missingUsage": "Incomplete usage; awaiting estimate",
  "subscription.usd.unavailable": "Estimate temporarily unavailable",
  "subscription.usd.resetAdjusted": "Only usage after the early reset",
}
