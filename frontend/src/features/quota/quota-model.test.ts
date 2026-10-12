import { describe, expect, it } from "vitest"
import { accounts, providers, quota, balance } from "./quota-test-helpers"
import {
  describeQuotaAccounts,
  makeQuotaEntry,
  filterQuotaEntries,
  quotaSummary,
  type QuotaSnapshot,
} from "./quota-model"

const items = describeQuotaAccounts(accounts, providers)
const base: QuotaSnapshot = { pending: false, fetching: false, error: null }
const now = Date.parse("2026-10-09T00:00:00Z")

describe("quota overview model", () => {
  it("classifies capability and authorization independently of account enablement", () => {
    expect(items[0]).toMatchObject({ kind: "subscription", canQuery: true })
    expect(items[2]).toMatchObject({ kind: "subscription", canQuery: false })
    expect(items[3]).toMatchObject({ kind: "balance", canQuery: true })
    expect(items[5]).toMatchObject({ kind: "unsupported", canQuery: false })
    expect(describeQuotaAccounts([accounts[0]!], [])[0]?.kind).toBe("unsupported")
  })
  it("requires complete valid windows before reporting availability or exhaustion", () => {
    expect(makeQuotaEntry(items[0]!, { ...base, quota }, now).status).toBe("normal")
    expect(
      makeQuotaEntry(items[0]!, { ...base, quota: { ...quota, fiveHour: null } }, now).status,
    ).toBe("unknown")
    expect(
      makeQuotaEntry(
        items[0]!,
        { ...base, quota: { ...quota, weekly: { remainingPercent: -1 } } },
        now,
      ).status,
    ).toBe("unknown")
    expect(
      makeQuotaEntry(
        items[0]!,
        { ...base, quota: { ...quota, weekly: { remainingPercent: 0 } } },
        now,
      ).status,
    ).toBe("exhausted")
  })
  it("marks passed resets and failed refreshes separately without losing snapshots", () => {
    const old = { ...quota, weekly: { remainingPercent: 0, resetsAt: "2020-01-01T00:00:00Z" } }
    expect(makeQuotaEntry(items[0]!, { ...base, quota: old }, now).status).toBe("stale")
    const failed = makeQuotaEntry(items[0]!, { ...base, quota, error: new Error("failed") }, now)
    expect(failed).toMatchObject({ status: "error", quota, fetchedAt: quota.fetchedAt })
    expect(makeQuotaEntry(items[2]!, { ...base, quota }, now).status).toBe("unauthenticated")
  })
  it("preserves official balances and distinguishes empty/null details from zero", () => {
    expect(makeQuotaEntry(items[3]!, { ...base, balance }, now).status).toBe("normal")
    expect(
      makeQuotaEntry(items[3]!, { ...base, balance: { ...balance, available: false } }, now).status,
    ).toBe("exhausted")
    for (const entries of [null, []])
      expect(
        makeQuotaEntry(items[3]!, { ...base, balance: { ...balance, entries } }, now).status,
      ).toBe("unknown")
  })
  it("does not report zero exhaustion without known results and filters identity/provider/status", () => {
    const pending = makeQuotaEntry(items[0]!, { ...base, pending: true }, now)
    expect(quotaSummary([pending])).toMatchObject({ exhausted: null, uncertain: 1 })
    const known = makeQuotaEntry(items[0]!, { ...base, quota }, now)
    expect(filterQuotaEntries([known, pending], " WORKSPACE-ONE ", "openai", "normal")).toEqual([
      known,
    ])
    expect(quotaSummary([known, pending])).toMatchObject({ exhausted: 0, uncertain: 1 })
  })
})
