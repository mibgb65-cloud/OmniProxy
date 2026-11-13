// 动态错误码只能映射到已存在的字面量键，避免把任意字符串交给 t()。
const IMPORT_ERROR_KEYS = {
  desktop_login_missing: "dumate.import.error.desktop_login_missing",
  desktop_login_invalid: "dumate.import.error.desktop_login_invalid",
  cookie_invalid: "dumate.import.error.cookie_invalid",
  sign_in_expired: "dumate.import.error.sign_in_expired",
  identity_changed: "dumate.import.error.identity_changed",
  not_dumate_provider: "dumate.import.error.not_dumate_provider",
  not_dumate_account: "dumate.import.error.not_dumate_account",
  invalid_account: "dumate.import.error.invalid_account",
  credential_unavailable: "dumate.import.error.credential_unavailable",
  console_unavailable: "dumate.import.error.console_unavailable",
  rate_limited: "dumate.import.error.rate_limited",
  invalid_response: "dumate.import.error.invalid_response",
  request_rejected: "dumate.import.error.request_rejected",
} as const

const REASON_KEYS = {
  sign_in_expired: "dumate.reason.sign_in_expired",
  console_unavailable: "dumate.reason.console_unavailable",
  invalid_response: "dumate.reason.invalid_response",
  rate_limited: "dumate.reason.rate_limited",
  request_rejected: "dumate.reason.request_rejected",
  credential_unavailable: "dumate.reason.credential_unavailable",
  verify_failed: "dumate.reason.verify_failed",
  claim_failed: "dumate.reason.claim_failed",
} as const

/** 后端错误文本形如 "dumate: cookie_invalid"；只提取稳定码。 */
export function dumateErrorCode(error: unknown): string {
  const message = error instanceof Error ? error.message : String(error ?? "")
  return /dumate:\s*([a-z_]+)/.exec(message)?.[1] ?? "operation_failed"
}

/** 导入失败的本地化文案键；未知码统一落到通用失败。 */
export function dumateImportErrorKey(code: string) {
  return (
    IMPORT_ERROR_KEYS[code as keyof typeof IMPORT_ERROR_KEYS] ??
    "dumate.import.error.operation_failed"
  )
}

/** 查询或签到失败的原因文案键；未知码统一落到未知错误。 */
export function dumateReasonKey(code: string | undefined) {
  return REASON_KEYS[(code ?? "") as keyof typeof REASON_KEYS] ?? "dumate.reason.operation_failed"
}
