import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react"
import { Events } from "@wailsio/runtime"
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
vi.mock("@wailsio/runtime", () => ({ Events: { On: vi.fn(() => () => {}) } }))
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
  total: 20,
  remaining: 16,
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
const usage = {
  inputTokens: "1234567",
  outputTokens: "12345",
  cachedInputTokens: "800000",
  requests: 5,
  unknownCacheRequests: 0,
  incompleteRequests: 0,
}

function setup(value: SubscriptionQuota | null = quota, fiveHour = true) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <SubscriptionQuotaUSD
          account={account}
          quota={value ?? undefined}
          fiveHour={fiveHour}
          now={now}
        />
      </QueryClientProvider>
    </I18nProvider>,
  )
  return client
}
beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(readSubscriptionQuotaEstimate).mockResolvedValue({
    fetchedAt: quota.fetchedAt,
    fiveHour: ready,
    weekly: { ...ready, total: 30, totalMin: 30, totalMax: 30 },
    usage,
  })
})

it("shows compact total and remaining columns without extra headings or reset notices", async () => {
  vi.mocked(readSubscriptionQuotaEstimate).mockResolvedValue({
    fetchedAt: quota.fetchedAt,
    fiveHour: { ...ready, resetAdjusted: true },
    weekly: { ...ready, total: 30, resetAdjusted: true },
    usage,
  })
  setup()
  const section = screen.getByRole("region", { name: "额度美元估算" })
  expect(await within(section).findByText("≈ $20.00")).toBeInTheDocument()
  expect(within(section).getByText("≈ $30.00")).toBeInTheDocument()
  expect(within(section).getAllByText("≈ $16.00")).toHaveLength(2)
  expect(within(section).getAllByRole("row")).toHaveLength(3)
  expect(within(section).getByRole("columnheader", { name: "预计总额" })).toBeInTheDocument()
  expect(within(section).getByRole("columnheader", { name: "预计剩余" })).toBeInTheDocument()
  expect(within(section).queryByText("额度美元估算")).not.toBeInTheDocument()
  expect(within(section).queryByText("仅统计提前重置后的用量")).not.toBeInTheDocument()
  expect(section.querySelector("details, [tabindex]")).toBeNull()
  expect(screen.queryByText("本机样本参考，非官方余额")).not.toBeInTheDocument()
})

it("does not invent a five-hour quota for weekly-only plans", async () => {
  setup(quota, false)
  expect(await screen.findByText("≈ $30.00")).toBeInTheDocument()
  expect(screen.queryByRole("rowheader", { name: "5h" })).not.toBeInTheDocument()
})

it("uses placeholders for missing prices and insufficient usage without extra notices", async () => {
  vi.mocked(readSubscriptionQuotaEstimate).mockResolvedValue({
    fetchedAt: quota.fetchedAt,
    weekly: { ...ready, status: "unknown_price", unpricedRequests: 2 },
    fiveHour: { ...ready, status: "insufficient_usage" },
    usage,
  })
  setup()
  await screen.findByText("1.23M")
  const section = screen.getByRole("region", { name: "额度美元估算" })
  expect(within(section).getAllByText("—")).toHaveLength(4)
  expect(screen.queryByText(/价格未知|待估算/)).not.toBeInTheDocument()
  expect(screen.queryByText(/≈ \$/)).not.toBeInTheDocument()
})

it("does not show estimates from another snapshot or after a reset", async () => {
  setup({ ...quota, weekly: { remainingPercent: 80, resetsAt: "2026-10-20T09:00:00Z" } })
  expect(await screen.findByText("≈ $20.00")).toBeInTheDocument()
  expect(
    within(screen.getByRole("region", { name: "额度美元估算" })).getAllByText("—"),
  ).toHaveLength(2)
  expect(screen.queryByText("≈ $30.00")).not.toBeInTheDocument()
})

it("does not render NaN or upstream errors as monetary values", async () => {
  vi.mocked(readSubscriptionQuotaEstimate).mockRejectedValue(new Error("private-estimate-marker"))
  const client = setup()
  await waitFor(() => expect(client.getQueryCache().getAll()[0]?.state.status).toBe("error"))
  expect(
    within(screen.getByRole("region", { name: "额度美元估算" })).getAllByText("—"),
  ).toHaveLength(4)
  expect(screen.queryByText(/private-estimate-marker|NaN/)).not.toBeInTheDocument()
})

it("shows retained input, output, and cache tokens without adding cache twice", async () => {
  setup()
  const section = screen.getByRole("region", { name: "Token 用量" })
  expect(await within(section).findByLabelText("1,234,567")).toHaveTextContent("1.23M")
  expect(within(section).getByText("12.35K")).toBeInTheDocument()
  expect(within(section).getByText("800K")).toBeInTheDocument()
  expect(within(section).queryByText("缓存已包含在输入中")).not.toBeInTheDocument()
  expect(within(section).queryByText("本机留存")).not.toBeInTheDocument()
})

it("reads tokens before any upstream quota is available and refreshes on local activity", async () => {
  vi.mocked(readSubscriptionQuotaEstimate).mockResolvedValue({ fetchedAt: "", usage })
  setup(null)
  expect(await screen.findByText("1.23M")).toBeInTheDocument()
  const call = vi.mocked(Events.On).mock.calls.find(([name]) => name === "activity:updated")
  expect(call).toBeDefined()
  const before = vi.mocked(readSubscriptionQuotaEstimate).mock.calls.length
  call?.[1]({ name: "activity:updated", data: "", sender: "" })
  await waitFor(() => expect(readSubscriptionQuotaEstimate).toHaveBeenCalledTimes(before + 1))
})

it("rejects stale snapshots", async () => {
  vi.mocked(readSubscriptionQuotaEstimate).mockResolvedValue({
    fetchedAt: "stale",
    weekly: ready,
    usage,
  })
  setup()
  expect(await screen.findByText("1.23M")).toBeInTheDocument()
  expect(screen.queryByText(/≈ \$/)).not.toBeInTheDocument()
})

it("skips local refreshes while hidden and catches up when visible", async () => {
  setup()
  await screen.findByText("1.23M")
  const before = vi.mocked(readSubscriptionQuotaEstimate).mock.calls.length
  const visibility = vi.spyOn(document, "visibilityState", "get").mockReturnValue("hidden")
  try {
    const call = vi.mocked(Events.On).mock.calls.find(([name]) => name === "activity:updated")
    call?.[1]({ name: "activity:updated", data: "", sender: "" })
    expect(readSubscriptionQuotaEstimate).toHaveBeenCalledTimes(before)
    visibility.mockReturnValue("visible")
    fireEvent(document, new Event("visibilitychange"))
    await waitFor(() => expect(readSubscriptionQuotaEstimate).toHaveBeenCalledTimes(before + 1))
  } finally {
    visibility.mockRestore()
  }
})

it.each([NaN, Infinity, -1])("does not display invalid monetary value %s", async (value) => {
  vi.mocked(readSubscriptionQuotaEstimate).mockResolvedValue({
    fetchedAt: quota.fetchedAt,
    weekly: { ...ready, total: value },
    usage,
  })
  setup(quota, false)
  expect(await screen.findByText("1.23M")).toBeInTheDocument()
  expect(
    within(screen.getByRole("region", { name: "额度美元估算" })).getAllByText("—"),
  ).toHaveLength(2)
  expect(screen.queryByText(/≈ \$|NaN|Infinity/)).not.toBeInTheDocument()
})
