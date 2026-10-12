import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import type { AccountItem } from "../../services/omniproxy"
import { readSubscriptionQuota, type SubscriptionQuota as Quota } from "../../services/subscription"
import { SubscriptionQuota } from "./subscription-quota"
import { useSubscriptionQuota } from "./use-subscription-quota"

vi.mock("../../services/subscription", async (original) => ({
  ...(await original<typeof import("../../services/subscription")>()),
  readSubscriptionQuota: vi.fn(),
}))
const account: AccountItem = {
  id: "acct_transition",
  providerId: "builtin_openai",
  name: "测试账号",
  authMethod: "subscription_login",
  authState: "authenticated",
  authAccountId: "identity-a",
  authPlan: "plus",
  enabled: true,
  createdAt: "2026-10-09T00:00:00Z",
  updatedAt: "2026-10-09T00:00:00Z",
}
const previousQuota: Quota = {
  plan: "plus",
  fiveHourEnabled: true,
  fiveHour: { remainingPercent: 75, resetsAt: "2099-10-09T00:00:00Z" },
  weekly: { remainingPercent: 40, resetsAt: "2099-10-15T00:00:00Z" },
  fetchedAt: "2026-10-09T00:00:00Z",
}
function Observer({ item }: { item: AccountItem }) {
  useSubscriptionQuota(item)
  return null
}
function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const view = (item: AccountItem) => (
    <I18nProvider>
      <QueryClientProvider client={client}>
        <SubscriptionQuota account={item} />
        <Observer item={item} />
      </QueryClientProvider>
    </I18nProvider>
  )
  const result = render(view(account))
  return { client, rerender: (item: AccountItem) => result.rerender(view(item)) }
}
describe("quota background refresh", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    vi.mocked(readSubscriptionQuota).mockResolvedValue(previousQuota)
  })
  it("retains numbers, timers and the same bar node when account metadata refreshes", async () => {
    const { rerender } = setup()
    const bar = await screen.findByRole("progressbar", { name: "5h 额度剩余额度" })
    const timer = screen.getAllByText(/后重置/)[0].textContent
    let finish!: (quota: Quota) => void
    vi.mocked(readSubscriptionQuota).mockImplementation(
      () =>
        new Promise((resolve) => {
          finish = resolve
        }),
    )
    rerender({ ...account, updatedAt: "2026-10-09T00:01:00Z" })
    await waitFor(() => expect(readSubscriptionQuota).toHaveBeenCalledTimes(2))
    expect(screen.getByText("75%")).toBeInTheDocument()
    expect(screen.getAllByText(/后重置/)[0]).toHaveTextContent(timer!)
    expect(screen.queryByText("时间待确认")).not.toBeInTheDocument()
    expect(screen.queryByText("正在查询额度…")).not.toBeInTheDocument()
    expect(screen.getByRole("progressbar", { name: "5h 额度剩余额度" })).toBe(bar)
    finish({ ...previousQuota, fiveHour: { ...previousQuota.fiveHour!, remainingPercent: 65 } })
    expect(await screen.findByText("65%")).toBeInTheDocument()
    expect(screen.getByRole("progressbar", { name: "5h 额度剩余额度" })).toBe(bar)
    expect(readSubscriptionQuota).toHaveBeenCalledTimes(2)
  })
  it("keeps the previous snapshot after a failed background refresh", async () => {
    const { rerender } = setup()
    await screen.findByText("75%")
    vi.mocked(readSubscriptionQuota).mockRejectedValue(new Error("subscription: quota_unavailable"))
    rerender({ ...account, updatedAt: "2026-10-09T00:02:00Z" })
    expect(await screen.findByRole("alert")).toHaveTextContent("额度查询失败")
    expect(screen.getByText("75%")).toBeInTheDocument()
    expect(screen.getByRole("progressbar", { name: "一周额度剩余额度" })).toHaveAttribute(
      "aria-valuenow",
      "40",
    )
  })
  it("does not reuse a previous identity or logged-out account snapshot", async () => {
    const { rerender } = setup()
    await screen.findByText("75%")
    vi.mocked(readSubscriptionQuota).mockImplementation(() => new Promise(() => {}))
    rerender({ ...account, authAccountId: "identity-b", updatedAt: "2026-10-09T00:03:00Z" })
    expect(screen.queryByText("75%")).not.toBeInTheDocument()
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument()
    rerender({ ...account, authState: "logged_out" })
    expect(screen.queryByText("75%")).not.toBeInTheDocument()
    expect(screen.getByText("登录后可查询额度。")).toBeInTheDocument()
  })
})
