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

it("distinguishes explicit zero, unknown details and incomplete requests", async () => {
  vi.mocked(readAccountTokenUsage).mockResolvedValue({
    ...usage,
    inputTokens: "0",
    outputTokens: "",
    cachedInputTokens: "",
    incompleteRequests: 1,
  })
  setup()
  expect(await screen.findByText("0")).toBeInTheDocument()
  expect(screen.getAllByText("—")).toHaveLength(2)
  expect(screen.getByText(/包含进行中或用量未完整返回/)).toBeInTheDocument()
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
