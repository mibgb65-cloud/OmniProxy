import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { readAccountBalance, type AccountBalanceInfo } from "../../services/balance"
import { useUIStore } from "../../stores/ui-store"
import { AccountBalance } from "./account-balance"

vi.mock("../../services/balance", async (original) => ({
  ...(await original<typeof import("../../services/balance")>()),
  readAccountBalance: vi.fn(),
}))
const account = {
  id: "kimi-test",
  providerId: "builtin_kimi",
  name: "Kimi",
  enabled: true,
  authMethod: "api_key",
  createdAt: "",
  updatedAt: "v1",
  kimi: { region: "cn", version: 1 },
}
const balance: AccountBalanceInfo = {
  available: true,
  entries: null,
  kimi: {
    currency: "CNY",
    available: "0.000000000000000001",
    voucher: "0.000000000000000001",
    cash: "-3.00001",
  },
  fetchedAt: "2026-10-15T00:00:00Z",
}
function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const view = (updatedAt: string) => (
    <I18nProvider>
      <QueryClientProvider client={client}>
        <AccountBalance account={{ ...account, updatedAt }} />
      </QueryClientProvider>
    </I18nProvider>
  )
  const result = render(view(account.updatedAt))
  return {
    user: userEvent.setup(),
    rerender: (updatedAt: string) => result.rerender(view(updatedAt)),
  }
}
beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(readAccountBalance).mockResolvedValue(balance)
})
it("keeps Kimi balances visible on failed refresh and isolates a new account version", async () => {
  const { user, rerender } = setup()
  await screen.findByText("¥0.000000000000000001")
  expect(screen.getByText("余额充足")).toBeInTheDocument()
  let reject!: (error: Error) => void
  vi.mocked(readAccountBalance).mockImplementationOnce(
    () =>
      new Promise((_, fail) => {
        reject = fail
      }),
  )
  await user.click(screen.getByRole("button", { name: "刷新余额" }))
  expect(screen.getByRole("button", { name: "刷新余额" })).toBeDisabled()
  expect(screen.getByText("¥0.000000000000000001")).toBeInTheDocument()
  reject(new Error("balance: service_unavailable"))
  expect(await screen.findByRole("alert")).toHaveTextContent("提供商余额服务暂时不可用")
  expect(screen.getByText("¥0.000000000000000001")).toBeInTheDocument()
  vi.mocked(readAccountBalance).mockImplementationOnce(() => new Promise(() => {}))
  rerender("v2")
  await waitFor(() => expect(readAccountBalance).toHaveBeenCalledTimes(3))
  expect(screen.queryByText("¥0.000000000000000001")).not.toBeInTheDocument()
})
it("displays an actual zero as insufficient without treating negative cash as available balance", async () => {
  vi.mocked(readAccountBalance).mockResolvedValue({
    ...balance,
    available: false,
    kimi: { ...balance.kimi!, available: "0.00000" },
  })
  setup()
  expect(await screen.findByText("¥0.00000")).toBeInTheDocument()
  expect(screen.getByText("余额不足")).toBeInTheDocument()
  expect(screen.getByText("-3.00001")).toBeInTheDocument()
})
it("shows a safe error without inventing a zero on the first failure", async () => {
  vi.mocked(readAccountBalance).mockRejectedValue(new Error("private-upstream-body"))
  setup()
  expect(await screen.findByRole("alert")).toHaveTextContent("余额查询失败")
  expect(screen.queryByText(/private-upstream-body|¥0/)).not.toBeInTheDocument()
})
it("uses the English labels for available, voucher and cash balances", async () => {
  useUIStore.setState({ language: "en" })
  setup()
  expect(await screen.findByText("Available balance")).toBeInTheDocument()
  expect(screen.getByText("Voucher balance")).toBeInTheDocument()
  expect(screen.getByText("Cash balance")).toBeInTheDocument()
})
