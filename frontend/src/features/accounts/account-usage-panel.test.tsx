import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import { act, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { readAccountTokenUsage } from "../../services/activity"
import type { AccountItem } from "../../services/omniproxy"
import { AccountDetailsCard } from "./account-details-card"
import { AccountDetailsList } from "./account-details-list"
import { AccountUsagePanel } from "./account-usage-panel"

vi.mock("../../services/activity", async (original) => ({
  ...(await original<typeof import("../../services/activity")>()),
  readAccountTokenUsage: vi.fn(),
}))
vi.mock("./account-key-preview", () => ({ AccountKeyPreview: () => null }))
vi.mock("@wailsio/runtime", () => ({ Events: { On: vi.fn(() => () => {}) } }))

const account: AccountItem = {
  id: "acct_0123456789abcdef0123456789abcdef",
  providerId: "builtin_river",
  name: "River test",
  authMethod: "api_key",
  enabled: true,
  createdAt: "2026-10-22T00:00:00Z",
  updatedAt: "2026-10-22T00:00:00Z",
}
const usage = {
  inputTokens: "1234567",
  outputTokens: "321",
  cachedInputTokens: "800000",
  requests: 2,
  unknownCacheRequests: 1,
  incompleteRequests: 0,
  pendingRequests: 0,
}
const actions = { busy: false, onToggle: vi.fn(), onDelete: vi.fn(), onEdit: vi.fn() }

function setup(view: "resource" | "cards" | "list" = "resource") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const rendered = render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        {view === "cards" ? (
          <AccountDetailsCard account={account} {...actions} />
        ) : view === "list" ? (
          <AccountDetailsList accounts={[account]} {...actions} />
        ) : (
          <AccountUsagePanel account={account} />
        )}
      </QueryClientProvider>
    </I18nProvider>,
  )
  return { ...rendered, client }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(readAccountTokenUsage).mockResolvedValue(usage)
})

it.each(["cards", "list"] as const)("replaces River balance placeholders in %s", async (view) => {
  setup(view)
  expect(await screen.findByText("1.23M")).toBeInTheDocument()
  expect(screen.getByText("321")).toBeInTheDocument()
  expect(screen.getByText("≥ 800K")).toBeInTheDocument()
  // 完整数字通过无障碍名称提供，不再使用系统原生标题提示。
  expect(screen.getByLabelText("1,234,567")).toBeInTheDocument()
  expect(screen.queryByText("余额")).not.toBeInTheDocument()
  expect(screen.queryByText("待接入")).not.toBeInTheDocument()
  expect(readAccountTokenUsage).toHaveBeenCalledWith(account.id)
})

it("distinguishes explicit zero and unknown details without a hint line", async () => {
  vi.mocked(readAccountTokenUsage).mockResolvedValue({
    ...usage,
    inputTokens: "0",
    outputTokens: "",
    cachedInputTokens: "",
    incompleteRequests: 1,
  })
  setup("cards")
  expect(await screen.findByText("0")).toBeInTheDocument()
  expect(screen.getAllByText("—")).toHaveLength(2)
  // 追加说明行会改变卡片高度并随请求反复闪烁，改为卡片头部的「正在使用」圆环。
  expect(screen.queryByText(/包含进行中或用量未完整返回/)).not.toBeInTheDocument()
})

it("shows an in-use ring while requests are pending and holds it briefly", async () => {
  vi.mocked(readAccountTokenUsage).mockResolvedValue({ ...usage, pendingRequests: 1 })
  const { unmount } = setup("cards")
  const ring = await screen.findByRole("status", { name: "该 API Key 正在使用中" })
  expect(ring).toBeInTheDocument()
  // 圆环固定占位，启用徽标的位置不随状态变化。
  expect(screen.getByText("已启用")).toBeInTheDocument()

  const listener = vi
    .mocked(Events.On)
    .mock.calls.find(([name]) => name === "activity:updated")?.[1]
  vi.mocked(readAccountTokenUsage).mockResolvedValue(usage)
  act(() => listener?.({ name: "activity:updated", data: [] }))
  await waitFor(
    () =>
      expect(
        screen.queryByRole("status", { name: "该 API Key 正在使用中" }),
      ).not.toBeInTheDocument(),
    { timeout: 3000 },
  )
  unmount()
})

it("explains the ring in a tooltip on hover", async () => {
  vi.mocked(readAccountTokenUsage).mockResolvedValue({ ...usage, pendingRequests: 1 })
  setup("cards")
  const ring = await screen.findByRole("status", { name: "该 API Key 正在使用中" })
  await userEvent.hover(ring)
  expect(await screen.findByRole("tooltip")).toHaveTextContent("该 API Key 正在使用中")
})

it("does not track accounts without the local usage panel", async () => {
  vi.mocked(readAccountTokenUsage).mockResolvedValue({ ...usage, pendingRequests: 1 })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <AccountDetailsCard account={{ ...account, providerId: "builtin_deepseek" }} {...actions} />
      </QueryClientProvider>
    </I18nProvider>,
  )
  await screen.findByText("已启用")
  expect(screen.queryByRole("status", { name: "该 API Key 正在使用中" })).not.toBeInTheDocument()
  expect(readAccountTokenUsage).not.toHaveBeenCalled()
})

it("hides raw errors and allows retrying local reads", async () => {
  vi.mocked(readAccountTokenUsage).mockRejectedValueOnce(new Error("private-river-marker"))
  setup()
  expect(await screen.findByRole("alert")).toHaveTextContent("无法读取本机用量")
  expect(screen.queryByText(/private-river-marker/)).not.toBeInTheDocument()
  await userEvent.click(screen.getByRole("button", { name: "重试" }))
  expect(await screen.findByText("1.23M")).toBeInTheDocument()
})

it("refreshes on activity and visibility and cleans up listeners", async () => {
  const off = vi.fn()
  vi.mocked(Events.On).mockReturnValueOnce(off)
  const { unmount } = setup()
  await screen.findByText("1.23M")
  const listener = vi
    .mocked(Events.On)
    .mock.calls.find(([name]) => name === "activity:updated")?.[1]
  expect(listener).toBeDefined()
  vi.mocked(readAccountTokenUsage).mockResolvedValue({ ...usage, inputTokens: "2000000" })
  act(() => listener?.({ name: "activity:updated", data: [] }))
  expect(await screen.findByText("2M")).toBeInTheDocument()
  const calls = vi.mocked(readAccountTokenUsage).mock.calls.length
  act(() => document.dispatchEvent(new Event("visibilitychange")))
  await waitFor(() => expect(readAccountTokenUsage).toHaveBeenCalledTimes(calls + 1))
  unmount()
  expect(off).toHaveBeenCalledOnce()
})
