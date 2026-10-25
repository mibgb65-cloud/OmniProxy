import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { Dialog } from "radix-ui"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { readMiniMaxQuota, openMiniMaxConsole } from "../../services/minimax"
import { AddAccountDialog } from "./add-account-dialog"
import { AccountEditForm } from "./account-edit-form"
import { MiniMaxBadge } from "./minimax-options"
import { MiniMaxResource } from "./minimax-resource"
import { describeQuotaAccounts, quotaEntryStatus } from "../quota/quota-model"

vi.mock("../../services/omniproxy", () => ({ getAccountKeyPreview: vi.fn(async () => "••••••••") }))
vi.mock("../../services/minimax", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../services/minimax")>()),
  readMiniMaxQuota: vi.fn(),
  openMiniMaxConsole: vi.fn(async () => {}),
}))
const provider = {
  id: "builtin_minimax",
  kind: "minimax",
  source: "builtin",
  name: "MiniMax",
  baseUrl: "",
  enabled: true,
  proxyMode: "inherit",
  createdAt: "",
  updatedAt: "",
}
const account = {
  id: "mini-account",
  providerId: provider.id,
  name: "MiniMax",
  enabled: true,
  authMethod: "api_key",
  createdAt: "",
  updatedAt: "now",
  minimax: { region: "global", billingMode: "subscription", version: 2 },
}
function Wrapper({ children }: { children: React.ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return (
    <I18nProvider>
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    </I18nProvider>
  )
}
beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(readMiniMaxQuota).mockResolvedValue({
    fetchedAt: "2026-10-15T00:00:00Z",
    buckets: [
      {
        model: "general",
        interval: { status: "known", remainingPercent: 13.5 },
        weekly: { status: "unknown" },
      },
    ],
  })
})

it("defaults new accounts to China payg and submits explicit options", async () => {
  const user = userEvent.setup(),
    save = vi.fn(async () => {}),
    batch = vi.fn(async () => {})
  render(
    <Wrapper>
      <AddAccountDialog provider={provider} onCreate={save} onBatchCreate={batch} />
    </Wrapper>,
  )
  await user.click(screen.getByRole("button", { name: "添加账号" }))
  expect(screen.getByLabelText("平台")).toHaveTextContent("国内")
  expect(screen.getByLabelText("Key 类型")).toHaveTextContent("普通按量 API")
  await user.type(screen.getByLabelText("API Key"), "placeholder-minimax-key")
  await user.click(screen.getByRole("button", { name: "保存账号" }))
  expect(save).toHaveBeenCalledWith({
    providerId: provider.id,
    name: "",
    apiKey: "placeholder-minimax-key",
    minimax: { region: "cn", billingMode: "payg", version: 0 },
  })
})

it("uses one selected region and subscription type for an entire batch", async () => {
  const user = userEvent.setup(),
    save = vi.fn(async () => {}),
    batch = vi.fn(async () => {})
  render(
    <Wrapper>
      <AddAccountDialog provider={provider} onCreate={save} onBatchCreate={batch} />
    </Wrapper>,
  )
  await user.click(screen.getByRole("button", { name: "添加账号" }))
  await user.click(screen.getByLabelText("平台"))
  await user.click(screen.getByRole("option", { name: "海外" }))
  await user.click(screen.getByLabelText("Key 类型"))
  await user.click(screen.getByRole("option", { name: "订阅 Key" }))
  await user.click(screen.getByRole("tab", { name: "批量添加" }))
  await user.type(screen.getByLabelText("API Keys"), "placeholder-one\nplaceholder-two")
  await user.click(screen.getByRole("button", { name: "保存全部账号" }))
  expect(batch).toHaveBeenCalledWith({
    providerId: provider.id,
    apiKeys: ["placeholder-one", "placeholder-two"],
    minimax: { region: "global", billingMode: "subscription", version: 0 },
  })
})

it("edits configuration with its version while keeping an empty replacement key", async () => {
  const user = userEvent.setup(),
    save = vi.fn(async () => {})
  render(
    <Wrapper>
      <Dialog.Root open>
        <AccountEditForm account={account} onSubmit={save} onRestoreFocus={() => {}} />
      </Dialog.Root>
    </Wrapper>,
  )
  await user.click(screen.getByLabelText("Key 类型"))
  await user.click(screen.getByRole("option", { name: "普通按量 API" }))
  await user.click(screen.getByRole("button", { name: "保存修改" }))
  expect(save).toHaveBeenCalledWith({
    id: account.id,
    name: account.name,
    apiKey: "",
    minimax: { region: "global", billingMode: "payg", version: 2 },
  })
})

it("requires legacy accounts to confirm billing rather than labelling them as subscription", () => {
  render(
    <Wrapper>
      <MiniMaxBadge
        account={{ ...account, minimax: { region: "global", billingMode: "", version: 1 } }}
      />
    </Wrapper>,
  )
  expect(screen.getByText("请编辑账号，确认平台和 Key 类型后再使用。")).toBeInTheDocument()
})

it("renders explicit official percentages, unknown windows and the matching console action", async () => {
  const user = userEvent.setup()
  render(
    <Wrapper>
      <MiniMaxResource account={account} />
    </Wrapper>,
  )
  await waitFor(() => expect(screen.getByText("13.5%")).toBeInTheDocument())
  expect(screen.getByText("未知")).toBeInTheDocument()
  expect(screen.getByRole("meter", { name: "当前窗口" })).toHaveAttribute("aria-valuenow", "13.5")
  expect(readMiniMaxQuota).toHaveBeenCalledTimes(1)
  await user.click(screen.getByRole("button", { name: "打开官方平台" }))
  expect(openMiniMaxConsole).toHaveBeenCalledWith(account.id)
  const item = describeQuotaAccounts([account], [provider])[0]
  expect(item.kind).toBe("minimax_plan")
  expect(item.canQuery).toBe(true)
  expect(
    quotaEntryStatus(
      item,
      { pending: false, fetching: false, error: null, minimax: await readMiniMaxQuota(account.id) },
      Date.now(),
    ),
  ).toBe("unknown")
})

it("keeps payg balance unknown and never sends a subscription query for it", async () => {
  render(
    <Wrapper>
      <MiniMaxResource
        account={{ ...account, minimax: { region: "cn", billingMode: "payg", version: 1 } }}
      />
    </Wrapper>,
  )
  expect(screen.getByText(/普通账户余额尚未接入/)).toBeInTheDocument()
  expect(readMiniMaxQuota).not.toHaveBeenCalled()
})
