import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import type { AccountItem } from "../../services/omniproxy"
import { readAccountBalance, type AccountBalanceInfo } from "../../services/balance"
import { AccountBalance } from "./account-balance"

vi.mock("../../services/balance", async (original) => ({
  ...(await original<typeof import("../../services/balance")>()),
  readAccountBalance: vi.fn(),
}))
const account: AccountItem = {
  id: "acct_balance",
  providerId: "builtin_deepseek",
  name: "余额测试",
  authMethod: "api_key",
  enabled: true,
  createdAt: "2026-10-09T00:00:00Z",
  updatedAt: "2026-10-09T00:00:00Z",
}
const balance: AccountBalanceInfo = {
  available: true,
  entries: [
    { currency: "CNY", total: "110.000000000000000001", granted: "10.00", toppedUp: "100.00" },
    { currency: "USD", total: "0.03", granted: "0.00", toppedUp: "0.03" },
  ],
  fetchedAt: "2026-10-09T00:00:00Z",
}
function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const view = (item: AccountItem) => (
    <I18nProvider>
      <QueryClientProvider client={client}>
        <AccountBalance account={item} />
      </QueryClientProvider>
    </I18nProvider>
  )
  const rendered = render(view(account))
  return { user: userEvent.setup(), rerender: (item: AccountItem) => rendered.rerender(view(item)) }
}
describe("DeepSeek balance", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    vi.mocked(readAccountBalance).mockResolvedValue(balance)
  })
  it("keeps official decimal precision and currencies separate", async () => {
    setup()
    expect(await screen.findByText("¥110.000000000000000001")).toBeInTheDocument()
    expect(screen.getByText("US$0.03")).toBeInTheDocument()
    expect(screen.getAllByText("赠送余额")).toHaveLength(2)
    expect(screen.getAllByText("充值余额")).toHaveLength(2)
    expect(screen.getByText("余额充足")).toBeInTheDocument()
  })
  it("preserves the last value while refreshing and after a failed refresh", async () => {
    const { user } = setup()
    await screen.findByText("¥110.000000000000000001")
    let reject!: (error: Error) => void
    vi.mocked(readAccountBalance).mockImplementation(
      () =>
        new Promise((_, fail) => {
          reject = fail
        }),
    )
    await user.click(screen.getByRole("button", { name: "刷新余额" }))
    expect(screen.getByText("¥110.000000000000000001")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "刷新余额" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "刷新余额" }).querySelector("svg")).toHaveClass(
      "animate-spin",
    )
    reject(new Error("balance: unauthorized"))
    expect(await screen.findByRole("alert")).toHaveTextContent("API Key 无效或已失效")
    expect(screen.getByRole("button", { name: "刷新余额" }).querySelector("svg")).not.toHaveClass(
      "animate-spin",
    )
    expect(screen.getByText("¥110.000000000000000001")).toBeInTheDocument()
  })
  it("does not display a fake zero for missing data or credential failure", async () => {
    vi.mocked(readAccountBalance).mockRejectedValue(new Error("private-key-error"))
    setup()
    expect(await screen.findByRole("alert")).toHaveTextContent("余额查询失败")
    expect(screen.queryByText(/private-key-error/)).not.toBeInTheDocument()
    expect(screen.queryByText("¥0.00")).not.toBeInTheDocument()
    expect(readAccountBalance).toHaveBeenCalledTimes(1)
  })
  it("uses a fresh cache after a metadata version change that may replace the key", async () => {
    const { rerender } = setup()
    await screen.findByText("¥110.000000000000000001")
    vi.mocked(readAccountBalance).mockImplementation(() => new Promise(() => {}))
    rerender({ ...account, updatedAt: "2026-10-09T01:00:00Z" })
    await waitFor(() => expect(readAccountBalance).toHaveBeenCalledTimes(2))
    expect(screen.queryByText("¥110.000000000000000001")).not.toBeInTheDocument()
  })
})
