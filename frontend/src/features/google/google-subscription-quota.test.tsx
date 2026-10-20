import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import type { AccountItem } from "../../services/omniproxy"
import {
  readGoogleSubscriptionQuota,
  readGoogleSubscriptionModels,
  type GoogleSubscriptionQuota as Quota,
} from "../../services/google-subscription"
import { GoogleSubscriptionQuota } from "./google-subscription-quota"
import { googleQuotaAccount, googleQuotaFixture } from "./google-quota-test-helpers"

vi.mock("../../services/google-subscription", () => ({
  readGoogleSubscriptionQuota: vi.fn(),
  readCachedGoogleSubscriptionQuota: vi.fn(async () => null),
  readGoogleSubscriptionModels: vi.fn(),
}))

function setup(account = googleQuotaAccount, count = 1) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  function Content({ item }: { item: AccountItem }) {
    return (
      <I18nProvider>
        <QueryClientProvider client={client}>
          {Array.from({ length: count }, (_, index) => (
            <GoogleSubscriptionQuota key={index} account={item} />
          ))}
        </QueryClientProvider>
      </I18nProvider>
    )
  }
  const view = render(<Content item={account} />)
  return { ...view, client, change: (item: AccountItem) => view.rerender(<Content item={item} />) }
}

describe("Google subscription quota", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(readGoogleSubscriptionModels).mockResolvedValue([])
    useUIStore.setState({ language: "zh" })
    vi.mocked(readGoogleSubscriptionQuota).mockResolvedValue(googleQuotaFixture)
  })
  it("renders separate official groups and four accessible quota windows", async () => {
    setup()
    const gemini = await screen.findByRole("region", { name: "Gemini 模型" })
    expect(
      within(gemini).getByRole("progressbar", { name: "Gemini 模型 · 5h 额度剩余额度" }),
    ).toHaveAttribute("aria-valuenow", "75")
    expect(within(gemini).getByText("80%")).toBeInTheDocument()
    expect(screen.getByRole("region", { name: "第三方模型（Claude / GPT）" })).toBeInTheDocument()
    expect(screen.getAllByRole("progressbar")).toHaveLength(4)
    expect(screen.queryByText(/暂未接入/)).not.toBeInTheDocument()
    expect(readGoogleSubscriptionQuota).toHaveBeenCalledWith(googleQuotaAccount.id)
  })
  it("preserves the last successful values when refresh fails without exposing the raw error", async () => {
    const user = userEvent.setup()
    setup()
    await screen.findByText("75%")
    vi.mocked(readGoogleSubscriptionQuota).mockRejectedValueOnce(
      new Error("google_subscription: upstream_unavailable private-token"),
    )
    await user.click(screen.getByRole("button", { name: "刷新 Google 订阅额度" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("无法连接 Google")
    expect(screen.getByText("75%")).toBeInTheDocument()
    expect(screen.queryByText(/private-token/)).not.toBeInTheDocument()
  })
  it("does not show another identity's cached values", async () => {
    const view = setup()
    await screen.findByText("75%")
    let finish!: (quota: Quota) => void
    vi.mocked(readGoogleSubscriptionQuota).mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve
        }),
    )
    view.change({ ...googleQuotaAccount, authAccountId: "2002" })
    await waitFor(() => expect(finish).toBeDefined())
    expect(screen.queryByText("75%")).not.toBeInTheDocument()
    await act(async () => finish({ groups: [], fetchedAt: "2026-10-16T00:01:00Z" }))
    expect(await screen.findByText("Google 暂未返回可用额度窗口。")).toBeInTheDocument()
  })
  it("keeps absent reset times unknown even when remaining quota is full", async () => {
    vi.mocked(readGoogleSubscriptionQuota).mockResolvedValue({
      groups: [
        {
          name: "Gemini Models",
          buckets: [{ id: "weekly", name: "", window: "weekly", remainingPercent: 100 }],
        },
      ],
      fetchedAt: "",
    })
    setup()
    await screen.findByText("100%")
    expect(screen.queryByText("未开始计时")).not.toBeInTheDocument()
    expect(screen.getByText("时间待确认")).toBeInTheDocument()
  })
  it("shares the same read between mounted views and skips logged-out accounts", async () => {
    const view = setup(googleQuotaAccount, 2)
    await waitFor(() => expect(screen.getAllByRole("progressbar")).toHaveLength(8))
    expect(readGoogleSubscriptionQuota).toHaveBeenCalledTimes(1)
    view.change({ ...googleQuotaAccount, authState: "logged_out" })
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "刷新 Google 订阅额度" })).not.toBeInTheDocument()
    expect(readGoogleSubscriptionQuota).toHaveBeenCalledTimes(1)
  })
})
