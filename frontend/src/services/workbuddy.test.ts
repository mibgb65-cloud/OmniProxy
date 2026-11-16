import { describe, expect, it } from "vitest"

import { workbuddyError, workbuddyErrorCodes } from "./workbuddy"

describe("workbuddyError", () => {
  it("recognises stable codes with their prefixes", () => {
    expect(workbuddyError(new Error("workbuddy_ai: sign_in_expired"))).toBe("sign_in_expired")
    expect(workbuddyError(new Error("subscription: busy"))).toBe("busy")
    // 登录会话把状态码直接交给界面，没有前缀。
    expect(workbuddyError("save_failed")).toBe("save_failed")
  })

  it("recognises core boundary errors with their prefix", () => {
    // proxy.Error 的文本是 `gateway: <code>`；只认 workbuddy 前缀会把额度失败吞成通用文案。
    expect(workbuddyError(new Error("gateway: upstream_unavailable"))).toBe("upstream_unavailable")
    expect(workbuddyError(new Error("gateway: rate_limited"))).toBe("rate_limited")
  })

  it("maps the account save failures added for the broken sign-in flow", () => {
    expect(workbuddyError(new Error("account: save workbuddy login"))).toBe("operation_failed")
    expect(workbuddyErrorCodes).toContain("save_failed")
    expect(workbuddyErrorCodes).toContain("invalid_input")
  })

  it("falls back for unknown or upstream text", () => {
    expect(workbuddyError(new Error("boom"))).toBe("operation_failed")
    expect(workbuddyError(undefined)).toBe("operation_failed")
  })
})
