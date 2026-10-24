import { describe, expect, it } from "vitest"

import { en } from "../../i18n/locales/en"
import { zh } from "../../i18n/locales/zh"
import { isKnownProviderKind, providerNameKeys } from "./provider-labels"

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

  it("keeps unknown kinds out of the label map", () => {
    expect(isKnownProviderKind("openai-compatible")).toBe(false)
    expect(isKnownProviderKind("unknown-provider")).toBe(false)
  })
})
