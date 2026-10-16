import type { GoogleSubscriptionQuota } from "../../services/google-subscription"

export function googleQuotaBuckets(quota?: GoogleSubscriptionQuota) {
  return (quota?.groups ?? []).flatMap((group) => group.buckets ?? [])
}

export function googleQuotaState(quota: GoogleSubscriptionQuota, now: number) {
  const buckets = googleQuotaBuckets(quota)
  if (
    !buckets.length ||
    buckets.some(
      (bucket) =>
        bucket.remainingPercent == null ||
        !Number.isFinite(bucket.remainingPercent) ||
        bucket.remainingPercent < 0 ||
        bucket.remainingPercent > 100 ||
        (bucket.resetsAt && !Number.isFinite(Date.parse(bucket.resetsAt))),
    )
  )
    return "unknown"
  if (buckets.some((bucket) => bucket.resetsAt && Date.parse(bucket.resetsAt) <= now))
    return "stale"
  return buckets.some((bucket) => bucket.remainingPercent === 0) ? "exhausted" : "normal"
}

export function googleQuotaPercent(value: number | null, language: string) {
  if (value == null || !Number.isFinite(value) || value < 0 || value > 100) return "—"
  const format = new Intl.NumberFormat(language, { maximumFractionDigits: 1 })
  // 向下显示避免不足满额被标为 100%，很小的正余额也不能写成耗尽。
  if (value > 0 && value < 0.1) return `<${format.format(0.1)}%`
  return `${format.format(Math.floor(value * 10) / 10)}%`
}
