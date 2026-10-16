import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactNode } from "react"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { describeQuotaAccounts } from "../quota/quota-model"
import { openKimiCodeConsole } from "../../services/kimi-settings"
import { AddAccountDialog } from "./add-account-dialog"
import { AccountDetailsCard } from "./account-details-card"
import { AccountDetailsList } from "./account-details-list"
import { readAccountBalance } from "../../services/balance"
import type { AccountItem, ProviderConfiguration } from "../../services/omniproxy"

vi.mock("../../services/kimi-settings", () => ({ openKimiCodeConsole: vi.fn() }))
vi.mock("../../services/balance", () => ({
  readAccountBalance: vi.fn(),
  balanceError: () => "query_failed",
}))
vi.mock("../../services/omniproxy", () => ({
  getAccountKeyPreview: vi.fn(async () => "masked"),
  copyAccountKey: vi.fn(),
}))
const provider: ProviderConfiguration = {
  id: "builtin_kimi",
  kind: "kimi",
  name: "Kimi",
  source: "builtin",
  baseUrl: "",
  enabled: true,
  createdAt: "",
  updatedAt: "",
}
const account: AccountItem = {
  id: "kimi-test",
  providerId: provider.id,
  name: "Coding",
  authMethod: "api_key",
  enabled: true,
  kimi: { region: "global", billingMode: "coding", version: 1 },
  createdAt: "2026-10-16T00:00:00Z",
  updatedAt: "2026-10-16T00:00:00Z",
}
beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(openKimiCodeConsole).mockResolvedValue(undefined)
})
function wrap(node: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <I18nProvider>
      <QueryClientProvider client={client}>{node}</QueryClientProvider>
    </I18nProvider>,
  )
}

it("submits a batch with the explicitly selected subscription and region", async () => {
  const user = userEvent.setup()
  const submit = vi.fn(async () => undefined)
  wrap(<AddAccountDialog provider={provider} onCreate={vi.fn()} onBatchCreate={submit} />)
  await user.click(screen.getByRole("button", { name: "添加账号" }))
  await user.click(screen.getByRole("combobox", { name: "计费方式" }))
  await user.click(screen.getByRole("option", { name: "Kimi Code 订阅" }))
  await user.click(screen.getByRole("combobox", { name: "Kimi 平台" }))
  await user.click(screen.getByRole("option", { name: "海外" }))
  await user.click(screen.getByRole("tab", { name: "批量添加" }))
  await user.type(
    screen.getByRole("textbox", { name: "API Keys" }),
    "placeholder-one\nplaceholder-two",
  )
  await user.click(screen.getByRole("button", { name: "保存全部账号" }))
  await waitFor(() =>
    expect(submit).toHaveBeenCalledWith(
      expect.objectContaining({ kimi: { region: "global", billingMode: "coding", version: 0 } }),
    ),
  )
})

it.each(["cards", "list"])(
  "shows unknown subscription quota without querying ordinary balance in %s",
  async (view) => {
    const props = {
      account,
      busy: false,
      onToggle: vi.fn(),
      onDelete: vi.fn(),
      onEdit: vi.fn(),
      balanceSupported: true,
    }
    wrap(
      view === "cards" ? (
        <AccountDetailsCard {...props} />
      ) : (
        <AccountDetailsList accounts={[account]} {...props} />
      ),
    )
    expect(await screen.findByText("Kimi Code 订阅额度")).toBeInTheDocument()
    expect(readAccountBalance).not.toHaveBeenCalled()
    expect(describeQuotaAccounts([account], [provider])[0]).toMatchObject({
      kind: "kimi_coding",
      canQuery: false,
    })
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "查看订阅控制台" }))
    expect(openKimiCodeConsole).toHaveBeenCalledWith(account.id)
  },
)
