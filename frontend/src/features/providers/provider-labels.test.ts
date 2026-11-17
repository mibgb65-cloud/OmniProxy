import { describe, expect, it } from "vitest"

import { en } from "../../i18n/locales/en"
import { zh } from "../../i18n/locales/zh"
import { isKnownProviderKind, providerNameKeys, subscriptionTypeKey } from "./provider-labels"

// 内置服务商新增 kind 时最容易漏掉的是界面名称：检查两套语言都有对应文案，
// 避免卡片直接显示 `providers.kind.xxx` 原始键。
describe("provider labels", () => {
  it("every built-in kind resolves in both locales", () => {
    for (const [kind, key] of Object.entries(providerNameKeys)) {
      expect(isKnownProviderKind(kind)).toBe(true)
      expect(en[key as keyof typeof en], `en ${kind}`).toBeTruthy()
      expect(zh[key as keyof typeof zh], `zh ${kind}`).toBeTruthy()
    }
  })

  it("includes the Oxion AI built-in", () => {
    expect(providerNameKeys.oxionai).toBe("providers.kind.oxionai")
    expect(en["providers.kind.oxionai"]).toBe("Oxion AI")
    expect(zh["providers.kind.oxionai"]).toBe("Oxion AI")
  })

  it("includes the HowToFix built-in", () => {
    expect(providerNameKeys.howtofix).toBe("providers.kind.howtofix")
    expect(en["providers.kind.howtofix"]).toBe("HowToFix")
    expect(zh["providers.kind.howtofix"]).toBe("HowToFix")
  })

  it("includes the WorkBuddy AI built-in", () => {
    expect(providerNameKeys.workbuddy_ai).toBe("providers.kind.workbuddy_ai")
    expect(en["providers.kind.workbuddy_ai"]).toBe("WorkBuddy AI")
    expect(zh["providers.kind.workbuddy_ai"]).toBe("WorkBuddy AI")
    expect(en["providers.subscriptionType.workbuddyAi"]).toBe("WorkBuddy AI subscription")
    expect(zh["providers.subscriptionType.workbuddyAi"]).toBe("WorkBuddy AI 订阅")
  })

  it("includes the WorkBuddy China built-in", () => {
    expect(providerNameKeys.workbuddy).toBe("providers.kind.workbuddy")
    expect(en["providers.kind.workbuddy"]).toBe("WorkBuddy (China)")
    expect(zh["providers.kind.workbuddy"]).toBe("WorkBuddy（国内版）")
    expect(en["providers.subscriptionType.workbuddy"]).toBe("WorkBuddy (China) subscription")
    expect(zh["providers.subscriptionType.workbuddy"]).toBe("WorkBuddy 国内版订阅")
  })

  it("resolves the subscription brand per provider", () => {
    expect(subscriptionTypeKey("builtin_workbuddy_ai")).toBe(
      "providers.subscriptionType.workbuddyAi",
    )
    expect(subscriptionTypeKey("workbuddy_ai")).toBe("providers.subscriptionType.workbuddyAi")
    expect(subscriptionTypeKey("builtin_workbuddy")).toBe("providers.subscriptionType.workbuddy")
    expect(subscriptionTypeKey("workbuddy")).toBe("providers.subscriptionType.workbuddy")
    expect(subscriptionTypeKey("builtin_google")).toBe("google.subscriptionType")
    expect(subscriptionTypeKey("builtin_openai")).toBe("subscription.type")
  })

  it("keeps unknown kinds out of the label map", () => {
    expect(isKnownProviderKind("openai-compatible")).toBe(false)
    expect(isKnownProviderKind("unknown-provider")).toBe(false)
  })
})
