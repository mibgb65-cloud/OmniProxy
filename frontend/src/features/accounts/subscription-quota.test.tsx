import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { readSubscriptionQuota } from "../../services/subscription"
import type { AccountItem } from "../../services/omniproxy"
import { SubscriptionQuota } from "./subscription-quota"

vi.mock("../../services/subscription", async (original) => ({
  ...(await original<typeof import("../../services/subscription")>()),
  readSubscriptionQuota: vi.fn(),
}))
const account: AccountItem = {
  id: "acct_quota",
  providerId: "builtin_openai",
  name: "额度账号",
  enabled: true,
  authMethod: "subscription_login",
  authState: "authenticated",
  authPlan: "plus",
  createdAt: "2026-10-08T00:00:00Z",
  updatedAt: "2026-10-08T00:00:00Z",
}
function setup(item = account) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <SubscriptionQuota account={item} />
      </QueryClientProvider>
    </I18nProvider>,
  )
  return userEvent.setup()
}
describe("subscription quota", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    vi.mocked(readSubscriptionQuota).mockResolvedValue({
      plan: "plus",
      fiveHourEnabled: true,
      fiveHour: { remainingPercent: 75 },
      weekly: { remainingPercent: 40 },
      fetchedAt: "2026-10-08T00:00:00Z",
    })
  })
  it("shows real remaining percentages for both Plus windows", async () => {
    setup()
    expect(await screen.findByRole("progressbar", { name: "5h 额度剩余额度" })).toHaveAttribute(
      "aria-valuenow",
      "75",
    )
    expect(screen.getByRole("progressbar", { name: "一周额度剩余额度" })).toHaveAttribute(
      "aria-valuenow",
      "40",
    )
    expect(screen.getByText("75%")).toBeInTheDocument()
    expect(screen.queryByText("50 / 50")).not.toBeInTheDocument()
  })
  it("shows only the weekly quota for Pro", async () => {
    vi.mocked(readSubscriptionQuota).mockResolvedValue({
      plan: "pro",
      fiveHourEnabled: false,
      weekly: { remainingPercent: 90 },
      fetchedAt: "2026-10-08T00:00:00Z",
    })
    setup({ ...account, authPlan: "pro" })
    expect(await screen.findByRole("progressbar", { name: "一周额度剩余额度" })).toHaveAttribute(
      "aria-valuenow",
      "90",
    )
    expect(screen.queryByText("5h 额度")).not.toBeInTheDocument()
  })
  it("does not imply full recovery after the returned reset time expires", async () => {
    vi.mocked(readSubscriptionQuota).mockResolvedValue({
      plan: "pro",
      fiveHourEnabled: false,
      weekly: { remainingPercent: 0, resetsAt: "2020-01-01T00:00:00Z" },
      fetchedAt: "2026-10-08T00:00:00Z",
    })
    setup({ ...account, authPlan: "pro" })
    expect(await screen.findByText("待刷新")).toBeInTheDocument()
    expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "0")
  })
  it("does not fabricate missing windows or retry failures automatically", async () => {
    vi.mocked(readSubscriptionQuota).mockResolvedValue({
      plan: "plus",
      fiveHourEnabled: true,
      fetchedAt: "2026-10-08T00:00:00Z",
    })
    setup()
    expect(await screen.findByText("部分额度窗口暂未返回。")).toBeInTheDocument()
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument()
  })
  it("shows a safe failure and supports an explicit retry", async () => {
    vi.mocked(readSubscriptionQuota).mockRejectedValue(
      new Error("subscription: quota_unavailable private-quota-value"),
    )
    const user = setup()
    expect(await screen.findByRole("alert")).toHaveTextContent("额度查询失败")
    expect(screen.queryByText(/private-quota-value/)).not.toBeInTheDocument()
    expect(readSubscriptionQuota).toHaveBeenCalledTimes(1)
    vi.mocked(readSubscriptionQuota).mockResolvedValue({
      plan: "plus",
      fiveHourEnabled: true,
      weekly: { remainingPercent: 80 },
      fetchedAt: "2026-10-08T00:00:00Z",
    })
    await user.click(screen.getByRole("button", { name: "刷新额度" }))
    await waitFor(() => expect(screen.queryByRole("alert")).not.toBeInTheDocument())
    expect(readSubscriptionQuota).toHaveBeenCalledTimes(2)
  })
  it("does not query or display cached quota for logged-out accounts", () => {
    setup({ ...account, authState: "logged_out" })
    expect(screen.getByText("登录后可查询额度。")).toBeInTheDocument()
    expect(readSubscriptionQuota).not.toHaveBeenCalled()
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument()
  })
})
