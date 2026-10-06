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
  type AccountItem,
  type ProviderConfiguration,
} from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import { ProviderAccountsView } from "./provider-accounts-view"

vi.mock("../../services/omniproxy", () => ({
  listProviders: vi.fn(),
  listAccounts: vi.fn(),
  createAccount: vi.fn(),
  createAccounts: vi.fn(),
  setAccountEnabled: vi.fn(),
  deleteAccount: vi.fn(),
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
  return render(<ProviderAccountsView providerId={providerId} />, { wrapper: Wrapper })
}

describe("ProviderAccountsView", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    vi.mocked(listProviders).mockResolvedValue([provider])
    vi.mocked(listAccounts).mockResolvedValue([account])
    vi.mocked(deleteAccount).mockResolvedValue(undefined)
  })

  it("shows account metadata and labels unavailable balance honestly", async () => {
    renderPage(provider.id)
    expect(await screen.findByRole("heading", { name: "DeepSeek 账号" })).toBeInTheDocument()
    expect(screen.getByRole("heading", { name: "主账号" })).toBeInTheDocument()
    expect(screen.getByText("余额")).toBeInTheDocument()
    expect(screen.getByText("待接入")).toBeInTheDocument()
    expect(screen.getByText("创建时间")).toBeInTheDocument()
    expect(screen.getByText("更新时间")).toBeInTheDocument()
    expect(screen.getByRole("switch", { name: "切换账号“主账号”的启用状态" })).toBeChecked()
    expect(screen.getByRole("button", { name: "添加账号" })).toBeInTheDocument()
    const deleteButton = screen.getByRole("button", { name: "删除账号“主账号”" })
    expect(deleteButton.textContent).toBe("")
    expect(deleteButton).toHaveAttribute("title", "删除")
    expect(screen.getByRole("link", { name: "返回账号" })).toHaveAttribute("href", "/accounts")
  })

  it("handles a deleted or unknown provider ID", async () => {
    renderPage("missing-provider")
    expect(await screen.findByRole("alert")).toHaveTextContent("未找到该提供商")
    expect(screen.queryByText("主账号")).not.toBeInTheDocument()
  })

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
})
