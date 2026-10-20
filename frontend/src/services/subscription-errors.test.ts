import { expect, it } from "vitest"
import {
  subscriptionError,
  subscriptionRefreshError,
  transientSubscriptionRefreshError,
} from "./subscription"

it.each([
  ["operation_timeout", "refresh_timeout", true],
  ["operation_cancelled", "refresh_cancelled", true],
  ["refresh_busy", "refresh_busy", true],
  ["request_failed", "refresh_unavailable", true],
  ["protocol_failed", "refresh_unavailable", true],
  ["account_id_pending", "account_id_pending", true],
  ["identity_changed", "identity_changed", false],
  ["codex_unavailable", "codex_unavailable", false],
  ["reauth_required", "reauth_required", false],
  ["account_update_failed", "account_update_failed", false],
])("%s 按刷新操作分类，同时保留真正需要处理的错误", (input, expected, transient) => {
  const error = new Error(`subscription: ${input}`)
  expect(subscriptionError(error)).toBe(input)
  const code = subscriptionRefreshError(error)
  expect(code).toBe(expected)
  expect(transientSubscriptionRefreshError(code)).toBe(transient)
})

it("未知刷新失败不渲染原始错误，也不要求重新授权", () => {
  const code = subscriptionRefreshError(new Error("private-authorization-value"))
  expect(code).toBe("refresh_unavailable")
  expect(transientSubscriptionRefreshError(code)).toBe(true)
})
