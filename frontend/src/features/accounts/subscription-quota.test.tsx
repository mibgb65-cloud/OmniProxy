import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, render, screen, waitFor } from "@testing-library/react"
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
  readCachedSubscriptionQuota: vi.fn(async () => null),
  readSubscriptionQuotaEstimate: vi.fn(async () => ({ fetchedAt: "" })),
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
  return { user: userEvent.setup(), client }
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
  it.each([
    ["quota_timeout", "额度查询超时，请检查网络后刷新。"],
    ["quota_busy", "账号操作繁忙，额度查询等待超时，请稍后刷新。"],
    ["quota_refresh_failed", "授权刷新失败，请检查网络后重试；若仍失败，请刷新账号状态。"],
  ])("explains %s without exposing upstream details", async (code, message) => {
    vi.mocked(readSubscriptionQuota).mockRejectedValue(
      new Error(`subscription: ${code} private-quota-marker`),
    )
    setup()
    expect(await screen.findByRole("alert")).toHaveTextContent(message)
    expect(screen.queryByText(/private-quota-marker/)).not.toBeInTheDocument()
    expect(readSubscriptionQuota).toHaveBeenCalledTimes(1)
  })
  it("shows a safe failure and supports an explicit retry", async () => {
    vi.mocked(readSubscriptionQuota).mockRejectedValue(
      new Error("subscription: quota_unavailable private-quota-value"),
    )
    const { user } = setup()
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
  it("瞬时失败时保留上次额度并标注，不再显示错误", async () => {
    const { client } = setup()
    expect(await screen.findByText("75%")).toBeInTheDocument()
    vi.mocked(readSubscriptionQuota).mockRejectedValueOnce(
      new Error("subscription: quota_timeout private-quota-marker"),
    )
    await act(async () => {
      await client.refetchQueries()
    })
    expect(readSubscriptionQuota).toHaveBeenCalledTimes(2)
    // 保留标记出现即说明重渲染已完成，随后的“无 alert”断言才有效。
    expect(await screen.findByText("已保留上次获取的信息")).toBeInTheDocument()
    expect(screen.queryByRole("alert")).not.toBeInTheDocument()
    expect(screen.getByText("75%")).toBeInTheDocument()
    expect(screen.getByRole("progressbar", { name: "5h 额度剩余额度" })).toHaveAttribute(
      "aria-valuenow",
      "75",
    )
    expect(screen.queryByText(/private-quota-marker/)).not.toBeInTheDocument()
  })
  it("确定性失败仍立即显示错误，不被旧数据掩盖", async () => {
    const { client } = setup()
    expect(await screen.findByText("75%")).toBeInTheDocument()
    vi.mocked(readSubscriptionQuota).mockRejectedValueOnce(
      new Error("subscription: quota_not_authenticated private-quota-marker"),
    )
    await act(async () => {
      await client.refetchQueries()
    })
    expect(await screen.findByRole("alert")).toHaveTextContent("请重新登录后查询额度。")
    expect(screen.queryByText("已保留上次获取的信息")).not.toBeInTheDocument()
    expect(screen.queryByText(/private-quota-marker/)).not.toBeInTheDocument()
  })
  it("does not query or display cached quota for logged-out accounts", () => {
    setup({ ...account, authState: "logged_out" })
    expect(screen.getByText("登录后可查询额度。")).toBeInTheDocument()
    expect(readSubscriptionQuota).not.toHaveBeenCalled()
    expect(screen.queryByRole("progressbar")).not.toBeInTheDocument()
  })
})
