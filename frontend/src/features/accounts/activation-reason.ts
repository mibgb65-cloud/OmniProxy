import type { TranslationKey } from "../../i18n/translation"

// 自动激活的稳定错误码 → 原因文案；未知码保持通用提示，不显示原始错误或账号身份。
const activationReasons: Record<string, TranslationKey> = {
  models_unavailable: "subscription.activation.reason.models",
  quota_unavailable: "subscription.activation.reason.quota",
  prepare_failed: "subscription.activation.reason.prepare",
  request_failed: "subscription.activation.reason.request",
  record_failed: "subscription.activation.reason.record",
  identity_changed: "subscription.activation.reason.identity",
  awaiting_quota: "subscription.activation.reason.timing",
  accounts_unavailable: "subscription.activation.reason.accounts",
}

/** 自动激活失败原因对应的文案键；未知码返回 null，由调用方使用通用文案。 */
export function activationReasonKey(code?: string): TranslationKey | null {
  if (!code) return null
  return activationReasons[code] ?? null
}
