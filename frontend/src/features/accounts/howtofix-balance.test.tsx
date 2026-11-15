import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import { beforeEach, expect, it, vi } from "vitest"
import type { Balance as HowToFixBalance } from "../../../bindings/omniproxy/internal/provider/howtofix/models"
import { I18nProvider } from "../../i18n/provider"
import { readAccountBalance, type AccountBalanceInfo } from "../../services/balance"
import { useUIStore } from "../../stores/ui-store"
import { AccountBalance } from "./account-balance"
import { HowToFixBalanceValues } from "./howtofix-balance"

vi.mock("../../services/balance", async (original) => ({
  ...(await original<typeof import("../../services/balance")>()),
  readAccountBalance: vi.fn(),
}))

const value: HowToFixBalance = {
  currency: "USD",
  remaining: "12.34",
}

function renderValues(balance: HowToFixBalance) {
  return render(
    <I18nProvider>
      <HowToFixBalanceValues value={balance} />
    </I18nProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
})

it("shows the prepaid remaining balance in Chinese", () => {
  renderValues(value)
  expect(screen.getByText("剩余额度")).toBeInTheDocument()
  expect(screen.getByText("US$12.34")).toBeInTheDocument()
})

it("shows an unknown marker instead of zero when the upstream omits the amount", () => {
  renderValues({ currency: "USD", remaining: "" })
  expect(screen.getByText("—")).toBeInTheDocument()
  // 未知金额不能被渲染成 0，否则会让人误以为已经用完；也不能出现「US$—」这种拼接。
  expect(screen.queryByText("0")).not.toBeInTheDocument()
  expect(screen.queryByText("US$—")).not.toBeInTheDocument()
})

it("uses the English label", () => {
  useUIStore.setState({ language: "en" })
  renderValues(value)
  expect(screen.getByText("Remaining")).toBeInTheDocument()
  expect(screen.getByText("US$12.34")).toBeInTheDocument()
})

it("renders the HowToFix balance returned for an account", async () => {
  const info: AccountBalanceInfo = {
    available: true,
    entries: null,
    howtofix: { currency: "USD", remaining: "9.50" },
    fetchedAt: "2026-11-15T00:00:00Z",
  }
  vi.mocked(readAccountBalance).mockResolvedValue(info)
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <AccountBalance
          account={{
            id: "howtofix-test",
            providerId: "builtin_howtofix",
            name: "HowToFix",
            enabled: true,
            authMethod: "api_key",
            createdAt: "",
            updatedAt: "v1",
          }}
        />
      </QueryClientProvider>
    </I18nProvider>,
  )
  expect(await screen.findByText("US$9.50")).toBeInTheDocument()
})
