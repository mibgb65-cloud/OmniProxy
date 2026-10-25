import { subscriptionError } from "../../services/subscription"

/**
 * 上次成功结果可继续沿用的时长。
 * 瞬时失败只在这段时间内沿用旧额度；超时后仍然透出错误，避免长期掩盖真实故障。
 */
export const QUOTA_KEEP_LAST_GOOD_MS = 10 * 60 * 1000

// 网络、超时与上游繁忙属于瞬时失败，可以沿用上次成功结果。
// quota_unavailable 就是用户反复看到的那句“额度查询失败，请稍后刷新。”：生产日志里它
// 对应 rate_limits 阶段的 quota_unavailable（上游 request_failed），属于瞬时失败；
// 确定性失败另有稳定码（quota_not_authenticated、identity_changed、invalid_account、
// codex_unavailable、quota_cache_unavailable 等），必须立即透出。
// quota_transient 是后端 HTTP 额度快路径的瞬时标记。
const transientQuotaCodes = [
  "quota_transient",
  "quota_unavailable",
  "quota_timeout",
  "quota_busy",
  "quota_refresh_failed",
  "operation_timeout",
] as const

export function isTransientQuotaError(error: unknown): boolean {
  const code: string = subscriptionError(error)
  return (transientQuotaCodes as readonly string[]).includes(code)
}

/**
 * keep-last-good：只有瞬时失败且上次成功仍在窗口内时，才用旧数据替代错误。
 * 组件因此不会在每次后台刷新失败时弹出红色提示。
 * `now` 由调用方给出，通常取失败发生的时刻（`errorUpdatedAt`），这样结果稳定且可复现。
 */
export function retainQuota<T>(input: {
  data: T | undefined
  dataUpdatedAt: number
  isError: boolean
  error: unknown
  now: number
  keepMs?: number
}): { data: T | undefined; error: unknown; retained: boolean } {
  const { data, dataUpdatedAt, isError, error, now, keepMs = QUOTA_KEEP_LAST_GOOD_MS } = input
  if (!isError) return { data, error: null, retained: false }
  const withinWindow = now - dataUpdatedAt < keepMs
  if (data !== undefined && withinWindow && isTransientQuotaError(error))
    return { data, error: null, retained: true }
  return { data, error, retained: false }
}
