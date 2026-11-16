import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type { CreditWindow, Quota } from "../../bindings/omniproxy/internal/application/workbuddy"

export type WorkBuddyQuota = Quota
export type WorkBuddyCreditWindow = CreditWindow

export const workbuddyQuota = async (id: string) => AppService.WorkBuddyQuota(id)
// 本机保留的快照只用于首屏，不发网络请求；没有快照时后端返回 null。
export const workbuddyCachedQuota = async (id: string) => AppService.WorkBuddyCachedQuota(id)

// 只识别稳定错误码：上游原文可能包含身份信息，不能渲染到界面。
// WorkBuddy 的授权/额度错误带 `workbuddy_ai:` 前缀，登录会话沿用 `subscription:` 前缀。
export const workbuddyErrorCodes = [
  "sign_in_expired",
  "unauthorized",
  "channel_rejected",
  "signature_invalid",
  "model_not_entitled",
  "user_agent_refused",
  "rate_limited",
  "upstream_unavailable",
  "invalid_response",
  "refresh_rejected",
  "credential_unavailable",
  "busy",
  "cancelled",
  "login_expired",
  "identity_changed",
  "configuration_changed",
  "save_failed",
  "invalid_input",
  "browser_failed",
] as const
export type WorkBuddyErrorCode = (typeof workbuddyErrorCodes)[number] | "operation_failed"

export function workbuddyError(error: unknown): WorkBuddyErrorCode {
  const text = error instanceof Error ? error.message : String(error)
  // 已是稳定码（登录会话直接返回码）时按原样识别，避免二次包装后丢失语义；
  // 其余按前缀匹配，忽略错误原文可能附带的上下文。
  const known = workbuddyErrorCodes.find(
    (code) =>
      text === code ||
      text.includes(`workbuddy_ai: ${code}`) ||
      text.includes(`subscription: ${code}`),
  )
  return known ?? "operation_failed"
}
