import { act, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { listAccounts, listProviders } from "../../services/omniproxy"
import { readSubscriptionQuota, type SubscriptionQuota } from "../../services/subscription"
import { readAccountBalance } from "../../services/balance"
import {
  readGoogleSubscriptionQuota,
  readGoogleSubscriptionModels,
} from "../../services/google-subscription"
import { googleQuotaFixture } from "../google/google-quota-test-helpers"
import { useUIStore } from "../../stores/ui-store"
import { selectOption } from "../../test/select-option"
import { renderQuota, resetQuotaServices, quota, accounts, providers } from "./quota-test-helpers"

vi.mock("../../services/omniproxy", async (original) => ({
  ...(await original<typeof import("../../services/omniproxy")>()),
  listAccounts: vi.fn(),
  listProviders: vi.fn(),
}))
vi.mock("../../services/subscription", async (original) => ({
  ...(await original<typeof import("../../services/subscription")>()),
  readSubscriptionQuota: vi.fn(),
}))
vi.mock("../../services/balance", async (original) => ({
  ...(await original<typeof import("../../services/balance")>()),
  readAccountBalance: vi.fn(),
}))
vi.mock("../../services/google-subscription", () => ({
  readGoogleSubscriptionQuota: vi.fn(),
  readGoogleSubscriptionModels: vi.fn(),
}))
vi.mock("@wailsio/runtime", async (original) => ({
  ...(await original<typeof import("@wailsio/runtime")>()),
  Events: { On: vi.fn(() => vi.fn()) },
}))

async function ready() {
  await waitFor(() => expect(screen.getByRole("button", { name: "刷新当前列表" })).toBeEnabled())
}

describe("quota overview", () => {
  it("labels Google subscriptions correctly without querying Codex quota", async () => {
    vi.mocked(readGoogleSubscriptionQuota).mockResolvedValue(googleQuotaFixture)
    vi.mocked(listAccounts).mockResolvedValue([
      { ...accounts[0], id: "google-sub", providerId: "builtin_google", authPlan: "g1-pro-tier" },
    ])
    vi.mocked(listProviders).mockResolvedValue([
      { ...providers[0], id: "builtin_google", kind: "google", name: "Google" },
    ])
    renderQuota()
    expect(await screen.findByText("Google / Antigravity")).toBeInTheDocument()
    expect(await screen.findByText("75%")).toBeInTheDocument()
    expect(readGoogleSubscriptionQuota).toHaveBeenCalledWith("google-sub")
    expect(screen.queryByText("ChatGPT / Codex")).not.toBeInTheDocument()
    expect(readSubscriptionQuota).not.toHaveBeenCalled()
  })
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(readGoogleSubscriptionModels).mockResolvedValue([])
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    resetQuotaServices()
  })

  it("groups official quotas and exact currency amounts, counts states and links account details", async () => {
    renderQuota()
    await ready()
    expect(screen.getByRole("heading", { name: "OpenAI" })).toBeInTheDocument()
    expect(screen.getByRole("heading", { name: "DeepSeek" })).toBeInTheDocument()
    expect(screen.getByText("110.000000000000000001")).toBeInTheDocument()
    expect(screen.getByText("总余额 · USD")).toBeInTheDocument()
    expect(screen.getByText("75%")).toBeInTheDocument()
    expect(screen.getByText("0%")).toBeInTheDocument()
    expect(screen.getByText("可监控账号").closest('[data-slot="card"]')).toHaveTextContent("5")
    expect(screen.getByText("查询异常").closest('[data-slot="card"]')).toHaveTextContent("1")
    expect(readSubscriptionQuota).toHaveBeenCalledTimes(2)
    expect(readSubscriptionQuota).not.toHaveBeenCalledWith("sub-out")
    expect(readAccountBalance).toHaveBeenCalledWith("balance-good")
    expect(screen.queryByText(/private balance detail/)).not.toBeInTheDocument()
    expect(
      screen.getByRole("link", { name: "查看 member@example.invalid 的账号详情" }),
    ).toHaveAttribute("href", "/accounts/openai")
  })

  it("filters account identity, provider and resource state without querying unsupported accounts", async () => {
    const user = userEvent.setup()
    renderQuota()
    await ready()
    const search = screen.getByRole("searchbox", { name: "搜索账号" })
    await user.type(search, "WORKSPACE-ONE")
    expect(screen.getByText("当前显示 1 / 6 个账号")).toBeInTheDocument()
    expect(screen.queryByText("balance-good")).not.toBeInTheDocument()
    await user.clear(search)
    await selectOption(user, screen.getByRole("combobox", { name: "提供商" }), "OpenAI")
    await selectOption(user, screen.getByRole("combobox", { name: "资源状态" }), "需要登录")
    expect(screen.getByText("当前显示 1 / 6 个账号")).toBeInTheDocument()
    expect(screen.getByRole("heading", { name: "out@example.invalid" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "刷新当前列表" })).toBeDisabled()
    expect(readSubscriptionQuota).toHaveBeenCalledTimes(2)
    expect(readAccountBalance).toHaveBeenCalledTimes(2)
  })

  it("queries Kimi's regional balance, counts exhaustion and skips unconfirmed platforms", async () => {
    vi.mocked(listProviders).mockResolvedValue([
      { ...providers[0]!, id: "builtin_kimi", kind: "kimi", name: "Kimi" },
    ])
    const account = {
      ...accounts[3]!,
      id: "kimi-known",
      providerId: "builtin_kimi",
      name: "Kimi CN",
      kimi: { region: "cn", version: 1 },
    }
    vi.mocked(listAccounts).mockResolvedValue([
      account,
      { ...account, id: "kimi-unknown", name: "Kimi unknown", kimi: undefined },
    ])
    vi.mocked(readAccountBalance).mockResolvedValue({
      available: false,
      entries: null,
      fetchedAt: "2026-10-15T00:00:00Z",
      kimi: { currency: "CNY", available: "0.00000", voucher: "0", cash: "-3.00001" },
    })
    renderQuota()
    await ready()
    expect(await screen.findByText("¥0.00000")).toBeInTheDocument()
    expect(screen.getByText("现金余额")).toBeInTheDocument()
    expect(screen.getByText("-3.00001")).toBeInTheDocument()
    expect(readAccountBalance).toHaveBeenCalledExactlyOnceWith("kimi-known")
    expect(screen.getByText("可监控账号").closest('[data-slot="card"]')).toHaveTextContent("1")
    expect(
      screen.queryByRole("button", { name: "刷新 Kimi unknown 的资源" }),
    ).not.toBeInTheDocument()
    expect(
      screen.getAllByText("额度耗尽／余额不足")[0]!.closest('[data-slot="card"]'),
    ).toHaveTextContent("1")
  })

  it("retains last quota and refresh time while a row refresh fails, then allows retry", async () => {
    const user = userEvent.setup()
    renderQuota()
    await ready()
    let fail!: (error: Error) => void
    vi.mocked(readSubscriptionQuota).mockImplementationOnce(
      () =>
        new Promise((_, reject) => {
          fail = reject
        }),
    )
    const button = screen.getByRole("button", { name: "刷新 member@example.invalid 的资源" })
    const card = within(button.closest('[data-slot="card"]')!)
    await user.click(button)
    expect(button).toBeDisabled()
    expect(card.getByText("75%")).toBeInTheDocument()
    expect(card.getByText(/最后成功刷新/)).toBeInTheDocument()
    await waitFor(() => expect(fail).toBeDefined())
    await act(async () => {
      fail(new Error("subscription: quota_unavailable private detail"))
    })
    expect(await card.findByRole("alert")).toHaveTextContent("额度查询失败")
    expect(card.getByText("75%")).toBeInTheDocument()
    expect(card.queryByText(/private detail/)).not.toBeInTheDocument()
    expect(button).toBeEnabled()
    await user.click(button)
    await waitFor(() => expect(card.queryByRole("alert")).not.toBeInTheDocument())
  })

  it("refreshes only eligible accounts in the filtered list and reports partial failures", async () => {
    const user = userEvent.setup()
    renderQuota()
    await ready()
    await selectOption(user, screen.getByRole("combobox", { name: "提供商" }), "OpenAI")
    vi.mocked(readSubscriptionQuota).mockImplementation(async (id) => {
      if (id === "sub-empty") throw new Error("subscription: quota_unavailable")
      return quota
    })
    await user.click(screen.getByRole("button", { name: "刷新当前列表" }))
    expect(await screen.findByText("已完成 2 个账号的刷新，1 个查询失败。")).toBeInTheDocument()
    expect(readSubscriptionQuota).toHaveBeenCalledTimes(4)
    expect(readAccountBalance).toHaveBeenCalledTimes(2)
    expect(readSubscriptionQuota).not.toHaveBeenCalledWith("sub-out")
  })

  it("cancels queued subscriptions when leaving the page", async () => {
    let finish!: (value: SubscriptionQuota) => void
    vi.mocked(readSubscriptionQuota).mockImplementation(
      () =>
        new Promise((resolve) => {
          finish = resolve
        }),
    )
    const { unmount } = renderQuota()
    await waitFor(() => expect(readSubscriptionQuota).toHaveBeenCalledTimes(1))
    unmount()
    await act(async () => {
      finish(quota)
    })
    expect(readSubscriptionQuota).toHaveBeenCalledTimes(1)
  })

  it("does not show fake resource counts after metadata failure and supports retry", async () => {
    const user = userEvent.setup()
    vi.mocked(listAccounts).mockRejectedValueOnce(new Error("private metadata"))
    renderQuota()
    expect(await screen.findByRole("alert")).toHaveTextContent("无法读取完整账号信息")
    expect(readSubscriptionQuota).not.toHaveBeenCalled()
    expect(readAccountBalance).not.toHaveBeenCalled()
    expect(screen.getByText("可监控账号").closest('[data-slot="card"]')).toHaveTextContent("—")
    await user.click(screen.getByRole("button", { name: "重新读取账号" }))
    await ready()
    expect(await screen.findByText("75%")).toBeInTheDocument()
    expect(listProviders).toHaveBeenCalled()
  })
})
