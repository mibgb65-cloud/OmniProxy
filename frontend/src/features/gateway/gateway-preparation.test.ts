import { describe, expect, it } from "vitest"
import type { AccountItem, ProviderConfiguration } from "../../services/omniproxy"
import { getConfiguredProxyAddress, getConfiguredResources } from "./gateway-preparation"

describe("网关接入准备", () => {
  it.each([undefined, 0, -1, 65536, 3001.5, Number.NaN, Number.POSITIVE_INFINITY])(
    "拒绝无效代理端口 %s",
    (port) => expect(getConfiguredProxyAddress(port)).toBeNull(),
  )

  it.each([1, 3001, 65535])("使用实际端口 %s 生成本机配置地址", (port) => {
    expect(getConfiguredProxyAddress(port)).toBe(`http://127.0.0.1:${port}`)
  })

  it("只统计账号与所属提供商均已启用的资源，不更改输入", () => {
    const providers: ProviderConfiguration[] = [
      { id: "enabled", enabled: true },
      { id: "disabled", enabled: false },
    ].map((item) => ({
      ...item,
      kind: "openai",
      source: "builtin",
      name: item.id,
      baseUrl: "",
      createdAt: "",
      updatedAt: "",
    }))
    const accounts: AccountItem[] = [
      { id: "included", providerId: "enabled", enabled: true },
      { id: "account-disabled", providerId: "enabled", enabled: false },
      { id: "provider-disabled", providerId: "disabled", enabled: true },
      { id: "orphan", providerId: "missing", enabled: true },
    ].map((item) => ({ ...item, name: item.id, createdAt: "", updatedAt: "" }))
    const before = JSON.stringify({ providers, accounts })

    expect(getConfiguredResources(providers, accounts)).toEqual({
      providerCount: 1,
      accountCount: 1,
    })
    expect(JSON.stringify({ providers, accounts })).toBe(before)
    expect(getConfiguredResources([], accounts)).toEqual({ providerCount: 0, accountCount: 0 })
  })
})
