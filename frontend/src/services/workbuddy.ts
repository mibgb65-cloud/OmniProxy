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
  // 错误可能来自多层：`workbuddy_ai: <code>`（适配器）、`subscription: <code>`（登录会话）、
  // `gateway: <code>`（核心边界）。按分隔符切词后匹配稳定码，避免被前缀差异吞成通用文案。
  const tokens = text.toLowerCase().split(/[^a-z_]+/)
  const known = workbuddyErrorCodes.find((code) => tokens.includes(code))
  return known ?? "operation_failed"
}
