import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import type { AccountItem } from "../../services/omniproxy"
import { readSubscriptionQuotaEstimate, type SubscriptionQuota } from "../../services/subscription"
import { SubscriptionQuotaUSD } from "./subscription-quota-usd"

vi.mock("../../services/subscription", async (original) => ({
  ...(await original<typeof import("../../services/subscription")>()),
  readSubscriptionQuotaEstimate: vi.fn(),
}))
const now = Date.parse("2026-10-20T10:00:00Z")
const account: AccountItem = {
  id: "acct_usd",
  providerId: "builtin_openai",
  name: "测试账号",
  enabled: true,
  authMethod: "subscription_login",
  authState: "authenticated",
  authEmail: "test@example.invalid",
  authAccountId: "test-workspace",
  createdAt: "",
  updatedAt: "",
}
const quota: SubscriptionQuota = {
  plan: "plus",
  fiveHourEnabled: true,
  fetchedAt: new Date(now).toISOString(),
  fiveHour: { remainingPercent: 90, resetsAt: "2026-10-20T13:00:00Z" },
  weekly: { remainingPercent: 80, resetsAt: "2026-10-23T10:00:00Z" },
}
const ready = {
  status: "ready",
  totalMin: 10,
  totalMax: 20,
  remainingMin: 8,
  remainingMax: 16,
  usedMin: 2,
  usedMax: 4,
  requests: 5,
  unpricedRequests: 0,
  resetAdjusted: false,
}

function setup(value = quota, fiveHour = true) {
  render(
    <I18nProvider>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <SubscriptionQuotaUSD account={account} quota={value} fiveHour={fiveHour} now={now} />
      </QueryClientProvider>
    </I18nProvider>,
  )
}
beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(readSubscriptionQuotaEstimate).mockResolvedValue({
    fetchedAt: quota.fetchedAt,
    fiveHour: ready,
    weekly: { ...ready, totalMin: 30, totalMax: 30 },
  })
})

it("shows 5h and weekly totals, remaining values, and a keyboard-accessible explanation", async () => {
  setup()
  const section = screen.getByRole("region", { name: "额度美元估算" })
  expect(await within(section).findByText("≈ $10.00–$20.00")).toBeInTheDocument()
  expect(within(section).getByText("≈ $30.00")).toBeInTheDocument()
  expect(within(section).getAllByText("预计剩余 ≈ $8.00–$16.00")).toHaveLength(2)
  await userEvent.click(within(section).getByText("本机样本参考，非官方余额"))
  expect(section.querySelector("details")).toHaveAttribute("open")
  expect(within(section).getByText(/至少已用 5%/)).toBeVisible()
})

it("does not invent a five-hour quota for weekly-only plans", async () => {
  setup(quota, false)
  expect(await screen.findByText("≈ $30.00")).toBeInTheDocument()
  expect(screen.queryByText("5h 预计总额")).not.toBeInTheDocument()
})

it("explains missing model prices instead of extrapolating a partial cost", async () => {
  vi.mocked(readSubscriptionQuotaEstimate).mockResolvedValue({
    fetchedAt: quota.fetchedAt,
    weekly: { ...ready, status: "unknown_price", unpricedRequests: 2 },
    fiveHour: { ...ready, status: "insufficient_usage" },
  })
  setup()
  expect(await screen.findByText("2 条请求价格未知")).toBeInTheDocument()
  expect(screen.getByText("已用不足 5%，待估算")).toBeInTheDocument()
  expect(screen.queryByText(/≈ \$/)).not.toBeInTheDocument()
})

it("does not show estimates from another snapshot or after a reset", async () => {
  setup({ ...quota, weekly: { remainingPercent: 80, resetsAt: "2026-10-20T09:00:00Z" } })
  expect(await screen.findByText("≈ $10.00–$20.00")).toBeInTheDocument()
  expect(screen.getByText("窗口已重置，待刷新")).toBeInTheDocument()
  expect(screen.queryByText("≈ $30.00")).not.toBeInTheDocument()
})

it("does not render NaN or upstream errors as monetary values", async () => {
  vi.mocked(readSubscriptionQuotaEstimate).mockRejectedValue(new Error("private-estimate-marker"))
  setup()
  expect(await screen.findAllByText("估算暂不可用")).toHaveLength(2)
  expect(screen.queryByText(/private-estimate-marker|NaN/)).not.toBeInTheDocument()
})
