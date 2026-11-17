import { describe, expect, it } from "vitest"

import { en } from "../../i18n/locales/en"
import { zh } from "../../i18n/locales/zh"
import { isWorkBuddyLoginKind, subscriptionLoginConfig } from "./subscription-login-config"

// 国内版是独立构建与独立服务商卡，但登录弹窗沿用 WorkBuddy 的等待与失败文案；
// 新增 kind 最容易漏的是文案键，这里同时检查两套语言。
describe("subscription login config", () => {
  it("keeps the China build separate but on the shared error dictionary", () => {
    const china = subscriptionLoginConfig("builtin_workbuddy")
    expect(china.kind).toBe("workbuddy")
    expect(china.brandKind).toBe("workbuddy")
    expect(china.deviceCode).toBe(false)
    expect(isWorkBuddyLoginKind(china.kind)).toBe(true)
    expect(isWorkBuddyLoginKind("google")).toBe(false)
    const keys = [
      china.titleKey,
      china.descriptionKey,
      china.storageKey,
      china.addKey,
      china.browserHintKey,
    ].filter((key): key is NonNullable<typeof key> => Boolean(key))
    for (const key of keys) {
      expect(en[key as keyof typeof en], `en ${key}`).toBeTruthy()
      expect(zh[key as keyof typeof zh], `zh ${key}`).toBeTruthy()
    }
    // 两个构建互不复用 providerId，避免国际版账号被当成国内版。
    expect(subscriptionLoginConfig("builtin_workbuddy_ai").kind).toBe("workbuddy_ai")
    expect(subscriptionLoginConfig("builtin_google").kind).toBe("google")
  })
})
