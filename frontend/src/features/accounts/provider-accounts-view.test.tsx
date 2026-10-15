import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterContextProvider,
} from "@tanstack/react-router"
import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import {
  listAccounts,
  deleteAccount,
  listProviders,
  getAccountKeyPreview,
  copyAccountKey,
  setAccountEnabled,
  type AccountItem,
  type ProviderConfiguration,
} from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import { ProviderAccountsView } from "./provider-accounts-view"
import { readAccountBalance } from "../../services/balance"

vi.mock("../../services/balance", () => ({
  readAccountBalance: vi.fn(async () => ({
    available: true,
    entries: [{ currency: "CNY", total: "110.00", granted: "10.00", toppedUp: "100.00" }],
    fetchedAt: "2026-10-09T00:00:00Z",
  })),
  balanceError: () => "query_failed",
}))

vi.mock("../../services/omniproxy", () => ({
  listProviders: vi.fn(),
  listAccounts: vi.fn(),
  createAccount: vi.fn(),
  createAccounts: vi.fn(),
  updateAccount: vi.fn(),
  setAccountEnabled: vi.fn(),
  deleteAccount: vi.fn(),
  getAccountKeyPreview: vi.fn(),
  copyAccountKey: vi.fn(),
}))

const provider: ProviderConfiguration = {
  id: "builtin_deepseek",
  kind: "deepseek",
  source: "builtin",
  name: "DeepSeek",
  baseUrl: "",
  enabled: true,
  createdAt: "2026-09-28T00:00:00Z",
  updatedAt: "2026-09-28T00:00:00Z",
}
const account: AccountItem = {
  id: "acct_0123456789abcdef0123456789abcdef",
  providerId: provider.id,
  name: "主账号",
  enabled: true,
  createdAt: "2026-09-28T01:00:00Z",
  updatedAt: "2026-09-28T02:00:00Z",
}

function renderPage(providerId: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const rootRoute = createRootRoute()
  const accountsRoute = createRoute({ getParentRoute: () => rootRoute, path: "/accounts" })
  const detailsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "/accounts/$providerId",
  })
  const router = createRouter({
    routeTree: rootRoute.addChildren([accountsRoute, detailsRoute]),
    history: createMemoryHistory({ initialEntries: [`/accounts/${providerId}`] }),
  })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <I18nProvider>
        <QueryClientProvider client={client}>
          <RouterContextProvider router={router}>{children}</RouterContextProvider>
        </QueryClientProvider>
      </I18nProvider>
    )
  }
  return {
    ...render(<ProviderAccountsView providerId={providerId} />, { wrapper: Wrapper }),
    client,
  }
}

describe("ProviderAccountsView", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useUIStore.setState({ language: "zh", accountView: "cards" })
    vi.mocked(listProviders).mockResolvedValue([provider])
    vi.mocked(listAccounts).mockResolvedValue([account])
    vi.mocked(deleteAccount).mockResolvedValue(undefined)
    vi.mocked(getAccountKeyPreview).mockResolvedValue("sk-t••••••••1234")
    vi.mocked(copyAccountKey).mockResolvedValue(undefined)
  })

  it("shows account metadata and live DeepSeek balance", async () => {
    renderPage(provider.id)
    expect(await screen.findByRole("heading", { name: "DeepSeek 账号" })).toBeInTheDocument()
    expect(screen.getByRole("heading", { name: "主账号" })).toBeInTheDocument()
    expect(await screen.findByText("¥110.00")).toBeInTheDocument()
    expect(screen.getByText("赠送余额")).toBeInTheDocument()
    expect(screen.getByText("充值余额")).toBeInTheDocument()
    expect(screen.queryByText("待接入")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "切换账号“主账号”的启用状态" })).toHaveAttribute(
      "aria-pressed",
      "true",
    )
    expect(screen.getByRole("button", { name: "添加账号" })).toBeInTheDocument()
    const deleteButton = screen.getByRole("button", { name: "删除账号“主账号”" })
    expect(deleteButton.textContent).toBe("")
    expect(deleteButton).not.toHaveAttribute("title")
    expect(screen.getByRole("link", { name: "返回账号" })).toHaveAttribute("href", "/accounts")
  })

  it("handles a deleted or unknown provider ID", async () => {
    renderPage("missing-provider")
    expect(await screen.findByRole("alert")).toHaveTextContent("未找到该提供商")
    expect(screen.queryByText("主账号")).not.toBeInTheDocument()
  })

  it.each(["cards", "list"] as const)(
    "shows a regional MiMo Token Plan with unknown credits in %s mode",
    async (accountView) => {
      useUIStore.setState({ accountView })
      vi.mocked(listProviders).mockResolvedValue([
        { ...provider, id: "builtin_mimo", kind: "mimo", name: "MiMo" },
      ])
      vi.mocked(listAccounts).mockResolvedValue([
        {
          ...account,
          providerId: "builtin_mimo",
          authMethod: "api_key",
          mimo: { region: "sgp", billingMode: "token_plan", version: 1 },
        },
      ])
      renderPage("builtin_mimo")
      expect(await screen.findByText("Token Plan · 新加坡")).toBeInTheDocument()
      expect(screen.getByText("Token Plan 套餐额度")).toBeInTheDocument()
      expect(screen.queryByText("充值余额")).not.toBeInTheDocument()
      expect(readAccountBalance).not.toHaveBeenCalled()
      expect(screen.getByRole("button", { name: "查看官方套餐" })).toBeEnabled()
    },
  )

  it.each(["cards", "list"] as const)(
    "shows Kimi's official cash and voucher balances in %s mode",
    async (accountView) => {
      useUIStore.setState({ accountView })
      vi.mocked(listProviders).mockResolvedValue([
        { ...provider, id: "builtin_kimi", kind: "kimi", name: "Kimi" },
      ])
      vi.mocked(listAccounts).mockResolvedValue([
        {
          ...account,
          providerId: "builtin_kimi",
          authMethod: "api_key",
          kimi: { region: "global", version: 1 },
        },
      ])
      vi.mocked(readAccountBalance).mockResolvedValueOnce({
        available: true,
        entries: null,
        fetchedAt: "2026-10-15T00:00:00Z",
        kimi: {
          currency: "USD",
          available: "1.000000000000000001",
          voucher: "1.000000000000000001",
          cash: "-3.00001",
        },
      })
      renderPage("builtin_kimi")
      expect(await screen.findByText("US$1.000000000000000001")).toBeInTheDocument()
      expect(screen.getByText("现金余额")).toBeInTheDocument()
      expect(screen.getByText("代金券余额")).toBeInTheDocument()
      expect(screen.getByText("-3.00001")).toBeInTheDocument()
      expect(screen.queryByText("充值余额")).not.toBeInTheDocument()
      expect(screen.queryByText("待接入")).not.toBeInTheDocument()
    },
  )

  it("opens a themed confirmation and cancels without deleting, restoring focus", async () => {
    const user = userEvent.setup()
    renderPage(provider.id)
    const removeButton = await screen.findByRole("button", { name: "删除账号“主账号”" })
    await user.click(removeButton)
    const dialog = screen.getByRole("alertdialog", { name: "删除账号" })
    expect(dialog).toHaveTextContent("确定删除“主账号”及其已保存的密钥吗？")
    const cancel = within(dialog).getByRole("button", { name: "取消" })
    expect(cancel).toHaveFocus()
    await user.click(cancel)
    expect(deleteAccount).not.toHaveBeenCalled()
    await waitFor(() => expect(removeButton).toHaveFocus())
    await user.click(removeButton)
    await user.keyboard("{Escape}")
    await waitFor(() => expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument())
    expect(deleteAccount).not.toHaveBeenCalled()
  })

  it("keeps deletion pending until the backend completes and blocks cancellation", async () => {
    let complete!: () => void
    vi.mocked(deleteAccount).mockImplementationOnce(
      () =>
        new Promise<void>((resolve) => {
          complete = resolve
        }),
    )
    const user = userEvent.setup()
    renderPage(provider.id)
    await user.click(await screen.findByRole("button", { name: "删除账号“主账号”" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    expect(screen.getByRole("button", { name: "正在删除…" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "取消" })).toBeDisabled()
    await user.keyboard("{Escape}")
    expect(screen.getByRole("alertdialog")).toBeInTheDocument()
    vi.mocked(listAccounts).mockResolvedValue([])
    complete()
    await waitFor(() => expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument())
    expect(deleteAccount).toHaveBeenCalledExactlyOnceWith(account.id)
  })

  it("keeps the confirmation open with an error and allows a retry", async () => {
    vi.mocked(deleteAccount).mockRejectedValueOnce(new Error("storage unavailable"))
    const user = userEvent.setup()
    renderPage(provider.id)
    await user.click(await screen.findByRole("button", { name: "删除账号“主账号”" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("删除失败")
    expect(screen.getByRole("alertdialog")).toBeInTheDocument()
    vi.mocked(listAccounts).mockResolvedValue([])
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    await waitFor(() => expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument())
    expect(deleteAccount).toHaveBeenCalledTimes(2)
  })

  it("switches views without changing accounts and remembers the selected mode", async () => {
    const user = userEvent.setup()
    renderPage(provider.id)
    await screen.findByRole("heading", { name: "主账号" })
    expect(screen.getByRole("radio", { name: "卡片展示" })).toBeChecked()
    await user.click(screen.getByRole("radio", { name: "列表展示" }))
    const table = screen.getByRole("table", { name: "账号信息列表" })
    expect(within(table).getByRole("rowheader", { name: /主账号/ })).toBeInTheDocument()
    expect(within(table).getByRole("columnheader", { name: "认证信息" })).toBeInTheDocument()
    expect(await within(table).findByText("sk-t••••••••1234")).toBeInTheDocument()
    expect(useUIStore.getState().accountView).toBe("list")
    expect(JSON.parse(localStorage.getItem("omniproxy-ui-state")!).state.accountView).toBe("list")
    await user.click(screen.getByRole("radio", { name: "卡片展示" }))
    expect(screen.queryByRole("table")).not.toBeInTheDocument()
    expect(screen.getByRole("heading", { name: "主账号" })).toBeInTheDocument()
    expect(deleteAccount).not.toHaveBeenCalled()
    expect(setAccountEnabled).not.toHaveBeenCalled()
  })

  it("keeps account controls and deletion confirmation working in list mode", async () => {
    useUIStore.setState({ accountView: "list" })
    const user = userEvent.setup()
    renderPage(provider.id)
    await screen.findByRole("table")
    vi.mocked(setAccountEnabled).mockResolvedValue({ ...account, enabled: false })
    vi.mocked(listAccounts).mockResolvedValue([{ ...account, enabled: false }])
    await user.click(screen.getByRole("button", { name: "切换账号“主账号”的启用状态" }))
    await waitFor(() => expect(setAccountEnabled).toHaveBeenCalledWith(account.id, false))
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "切换账号“主账号”的启用状态" })).toHaveAttribute(
        "aria-pressed",
        "false",
      ),
    )
    await user.click(screen.getByRole("button", { name: "删除账号“主账号”" }))
    const dialog = screen.getByRole("alertdialog", { name: "删除账号" })
    await user.click(within(dialog).getByRole("button", { name: "取消" }))
    expect(deleteAccount).not.toHaveBeenCalled()
  })

  it("copies through the desktop binding without receiving a full key", async () => {
    const user = userEvent.setup()
    const { client } = renderPage(provider.id)
    await screen.findByText("sk-t••••••••1234")
    await user.click(screen.getByRole("button", { name: "复制账号“主账号”的 API Key" }))
    await waitFor(() => expect(copyAccountKey).toHaveBeenCalledExactlyOnceWith(account.id))
    expect(await screen.findByRole("button", { name: "已复制" })).toBeInTheDocument()
    expect(client.getQueryData(["account-key-preview", account.id])).toBe("sk-t••••••••1234")
  })

  it("shows key-read and clipboard failures without exposing key contents", async () => {
    vi.mocked(getAccountKeyPreview).mockRejectedValueOnce(new Error("unavailable"))
    const user = userEvent.setup()
    renderPage(provider.id)
    expect(await screen.findByText("读取失败")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "重新读取密钥信息" }))
    await screen.findByText("sk-t••••••••1234")
    vi.mocked(copyAccountKey).mockRejectedValueOnce(new Error("clipboard unavailable"))
    await user.click(screen.getByRole("button", { name: "复制账号“主账号”的 API Key" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("复制失败")
  })
})
