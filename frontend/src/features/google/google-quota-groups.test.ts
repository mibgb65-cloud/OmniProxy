import { describe, expect, it } from "vitest"
import type { AccountItem } from "../../services/omniproxy"
import type { GoogleSubscriptionQuota } from "../../services/google-subscription"
import { buildQuotaGroupSections, quotaGroupSignature } from "./google-quota-groups"

function account(id: string, authMethod: "subscription_login" | "api_key" = "subscription_login") {
  const item: AccountItem = {
    id,
    providerId: "builtin_google",
    name: id,
    enabled: true,
    authMethod,
    createdAt: "2026-11-01T00:00:00Z",
    updatedAt: "2026-11-01T00:00:00Z",
  }
  return item
}

function quota(names: string[]): GoogleSubscriptionQuota {
  return {
    fetchedAt: "2026-11-19T00:00:00Z",
    groups: names.map((name) => ({ name, buckets: [] })),
  }
}

const baseGroups = ["Gemini Models", "Claude and GPT models"]

describe("quotaGroupSignature", () => {
  it("keeps the upstream order, trims and drops duplicates", () => {
    expect(
      quotaGroupSignature(quota([" Gemini Models ", "Claude and GPT models", "Gemini Models"])),
    ).toEqual(["Gemini Models", "Claude and GPT models"])
  })

  it("treats missing, empty and unnamed groups as unknown", () => {
    expect(quotaGroupSignature(undefined)).toBeNull()
    expect(quotaGroupSignature(quota([]))).toBeNull()
    expect(quotaGroupSignature(quota(["", "   "]))).toBeNull()
  })
})

describe("buildQuotaGroupSections", () => {
  it("separates accounts that carry an extra quota group and puts them first", () => {
    const plain = account("plain")
    const argon = account("argon")
    const sections = buildQuotaGroupSections(
      [plain, argon],
      new Map([
        ["plain", quota(baseGroups)],
        ["argon", quota([...baseGroups, "Gemini 4 Argon"])],
      ]),
    )
    expect(sections.map((section) => section.names)).toEqual([
      [...baseGroups, "Gemini 4 Argon"],
      baseGroups,
    ])
    expect(sections.map((section) => section.accounts.map((item) => item.id))).toEqual([
      ["argon"],
      ["plain"],
    ])
  })

  it("keeps accounts with the same group set together regardless of upstream order", () => {
    const sections = buildQuotaGroupSections(
      [account("left"), account("right")],
      new Map([
        ["left", quota(baseGroups)],
        ["right", quota([...baseGroups].reverse())],
      ]),
    )
    expect(sections).toHaveLength(1)
    expect(sections[0].accounts.map((item) => item.id)).toEqual(["left", "right"])
  })

  it("keeps the account order inside a section", () => {
    const sections = buildQuotaGroupSections(
      [account("second", "api_key"), account("first", "api_key")],
      new Map(),
    )
    expect(sections).toHaveLength(1)
    expect(sections[0].kind).toBe("apiKey")
    expect(sections[0].accounts.map((item) => item.id)).toEqual(["second", "first"])
  })

  it("puts API Key accounts next and unknown quota last without inventing groups", () => {
    const sections = buildQuotaGroupSections(
      [account("unknown"), account("key", "api_key"), account("known")],
      new Map([["known", quota(baseGroups)]]),
    )
    expect(sections.map((section) => section.kind)).toEqual(["groups", "apiKey", "unknown"])
    expect(sections[1].names).toBeNull()
    expect(sections[2].names).toBeNull()
    expect(sections[2].accounts.map((item) => item.id)).toEqual(["unknown"])
  })

  it("returns no sections when there are no accounts", () => {
    expect(buildQuotaGroupSections([], new Map())).toEqual([])
  })
})
