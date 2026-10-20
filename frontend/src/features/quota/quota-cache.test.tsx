import { act, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { readSubscriptionQuota } from "../../services/subscription"
import { readAccountBalance } from "../../services/balance"
import type { AccountBalanceInfo } from "../../services/balance"
import type { SubscriptionQuota } from "../../services/subscription"
import { useUIStore } from "../../stores/ui-store"
import { SubscriptionQuota as AccountQuota } from "../accounts/subscription-quota"
import { AccountBalance } from "../accounts/account-balance"
import { accounts, balance, quota, renderQuota, resetQuotaServices } from "./quota-test-helpers"

vi.mock("../../services/omniproxy", async (original) => ({
  ...(await original<typeof import("../../services/omniproxy")>()),
  listAccounts: vi.fn(),
  listProviders: vi.fn(),
}))
vi.mock("../../services/subscription", async (original) => ({
  ...(await original<typeof import("../../services/subscription")>()),
  readSubscriptionQuota: vi.fn(),
  readCachedSubscriptionQuota: vi.fn(async () => null),
}))
vi.mock("../../services/balance", async (original) => ({
  ...(await original<typeof import("../../services/balance")>()),
  readAccountBalance: vi.fn(),
}))
vi.mock("@wailsio/runtime", async (original) => ({
  ...(await original<typeof import("@wailsio/runtime")>()),
  Events: { On: vi.fn(() => vi.fn()) },
}))

describe("quota overview shared cache", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    resetQuotaServices()
  })
  it("shares requests with mounted account detail quota and balance views", async () => {
    renderQuota(
      <>
        <AccountQuota account={accounts[0]!} />
        <AccountBalance account={accounts[3]!} />
      </>,
    )
    await waitFor(() => expect(screen.getByRole("button", { name: "刷新当前列表" })).toBeEnabled())
    expect(readSubscriptionQuota).toHaveBeenCalledTimes(2)
    expect(readAccountBalance).toHaveBeenCalledTimes(2)
    expect(screen.getAllByText("75%")).toHaveLength(2)
    expect(screen.getByText("¥110.000000000000000001")).toBeInTheDocument()
  })
  it("keeps same-identity quota during metadata refresh but isolates changed credentials", async () => {
    const { client } = renderQuota()
    await waitFor(() => expect(screen.getByRole("button", { name: "刷新当前列表" })).toBeEnabled())
    let finishQuota!: (data: SubscriptionQuota) => void
    let finishBalance!: (data: AccountBalanceInfo) => void
    vi.mocked(readSubscriptionQuota).mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishQuota = resolve
        }),
    )
    vi.mocked(readAccountBalance).mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishBalance = resolve
        }),
    )
    await act(async () => {
      client.setQueryData(
        ["accounts"],
        accounts.map((item) =>
          ["sub-good", "balance-good"].includes(item.id)
            ? { ...item, updatedAt: "2026-10-09T01:00:00Z" }
            : item,
        ),
      )
    })
    await waitFor(() => expect(finishQuota).toBeDefined())
    await waitFor(() => expect(finishBalance).toBeDefined())
    expect(screen.getByText("75%")).toBeInTheDocument()
    expect(screen.queryByText("110.000000000000000001")).not.toBeInTheDocument()
    await act(async () => {
      finishQuota(quota)
      finishBalance(balance)
    })
    await waitFor(() => expect(screen.getByRole("button", { name: "刷新当前列表" })).toBeEnabled())
    expect(readSubscriptionQuota).toHaveBeenCalledTimes(3)
    expect(readAccountBalance).toHaveBeenCalledTimes(3)
  })
})
