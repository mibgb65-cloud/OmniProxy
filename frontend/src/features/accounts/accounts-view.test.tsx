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
  createAccount,
  deleteAccount,
  getProviderCatalog,
  listAccounts,
  listProviders,
  setAccountEnabled,
  type AccountItem,
  type ProviderConfiguration,
  type ProviderDefinition,
} from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import { AccountsView } from "./accounts-view"

vi.mock("../../services/omniproxy", () => ({
  getProviderCatalog: vi.fn(),
  listProviders: vi.fn(),
  listAccounts: vi.fn(),
  createAccount: vi.fn(),
  createAccounts: vi.fn(),
  updateAccount: vi.fn(),
  getAccountKeyPreview: vi.fn(async () => "••••••••"),
  copyAccountKey: vi.fn(),
  setAccountEnabled: vi.fn(),
  deleteAccount: vi.fn(),
}))

const kinds = [
  "openai",
  "anthropic",
  "google",
  "deepseek",
  "zhipu",
  "kimi",
  "minimax",
  "mimo",
  "stepfun",
]
const catalog: ProviderDefinition[] = kinds.map((kind) => ({
  kind,
  name: kind,
  defaultBaseUrl: "https://example.test",
  models: [],
  authMethods: ["api_key"],
  plannedAuthMethods: kind === "openai" ? ["subscription_login"] : null,
}))
const providers: ProviderConfiguration[] = kinds.map((kind) => ({
  id: `builtin_${kind}`,
  kind,
  source: "builtin",
  name: kind,
  baseUrl: "",
  enabled: true,
  proxyMode: "inherit",
  createdAt: "2026-09-28T00:00:00Z",
  updatedAt: "2026-09-28T00:00:00Z",
}))
const custom: ProviderConfiguration = {
  id: "prov_0123456789abcdef0123456789abcdef",
  kind: "openai-compatible",
  source: "custom",
  name: "Custom API",
  baseUrl: "https://example.test/v1",
  enabled: true,
  proxyMode: "inherit",
  createdAt: "2026-09-28T00:00:00Z",
  updatedAt: "2026-09-28T00:00:00Z",
}
let saved: AccountItem[] = []
let nextId = 0

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const rootRoute = createRootRoute()
  const accountsRoute = createRoute({ getParentRoute: () => rootRoute, path: "/accounts" })
  const detailsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "/accounts/$providerId",
  })
  const router = createRouter({
    routeTree: rootRoute.addChildren([accountsRoute, detailsRoute]),
    history: createMemoryHistory({ initialEntries: ["/accounts"] }),
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
  return { ...render(<AccountsView />, { wrapper: Wrapper }), router }
}

describe("AccountsView", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    saved = []
    nextId = 0
    vi.mocked(getProviderCatalog).mockResolvedValue(catalog)
    vi.mocked(listProviders).mockResolvedValue([...providers, custom])
    vi.mocked(listAccounts).mockImplementation(async () => saved)
    vi.mocked(createAccount).mockImplementation(async (input) => {
      const item: AccountItem = {
        id: `acct_${String(++nextId).padStart(32, "0")}`,
        providerId: input.providerId,
        name: input.name,
        enabled: true,
        createdAt: "2026-09-28T00:00:00Z",
        updatedAt: "2026-09-28T00:00:00Z",
      }
      saved = [item, ...saved]
      return item
    })
    vi.mocked(setAccountEnabled).mockImplementation(async (id, enabled) => {
      const item = { ...saved.find((account) => account.id === id)!, enabled }
      saved = saved.map((account) => (account.id === id ? item : account))
      return item
    })
    vi.mocked(deleteAccount).mockImplementation(async (id) => {
      saved = saved.filter((account) => account.id !== id)
    })
  })

  it("shows all existing providers and manages API Key accounts without displaying keys", async () => {
    const user = userEvent.setup()
    renderPage()
    expect((await screen.findAllByRole("button", { name: "添加账号" })).length).toBe(10)
    expect(screen.getByRole("heading", { name: "Custom API" })).toBeInTheDocument()
    expect(screen.queryByText("订阅登录待接入；目前可添加 API Key 账号。")).not.toBeInTheDocument()
    await user.click(screen.getAllByRole("button", { name: "添加账号" })[0])
    expect(screen.getByRole("dialog", { name: "添加 OpenAI 账号" })).toBeInTheDocument()
    await user.type(screen.getByLabelText("账号名称"), "主账号")
    await user.type(screen.getByLabelText("API Key"), "test-secret-not-real")
    await user.click(screen.getByRole("button", { name: "保存账号" }))
    await waitFor(() =>
      expect(createAccount).toHaveBeenCalledWith({
        providerId: "builtin_openai",
        name: "主账号",
        apiKey: "test-secret-not-real",
      }),
    )
    const account = await screen.findByText("主账号")
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    expect(within(account.closest("li")!).getByText("API Key · 已启用")).toBeInTheDocument()
    expect(screen.queryByText("test-secret-not-real")).not.toBeInTheDocument()
    expect(screen.queryByDisplayValue("test-secret-not-real")).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "切换账号“主账号”的启用状态" }))
    await waitFor(() => expect(setAccountEnabled).toHaveBeenCalledWith(saved[0].id, false))
    const deleteButton = screen.getByRole("button", { name: "删除账号“主账号”" })
    expect(deleteButton.textContent).toBe("")
    expect(deleteButton).not.toHaveAttribute("title")
    await user.click(deleteButton)
    const confirmation = screen.getByRole("alertdialog", { name: "删除账号" })
    expect(deleteAccount).not.toHaveBeenCalled()
    await user.click(within(confirmation).getByRole("button", { name: "确认删除" }))
    await waitFor(() => expect(deleteAccount).toHaveBeenCalled())
    await waitFor(() => expect(screen.queryByText("主账号")).not.toBeInTheDocument())
  })

  it("rejects empty keys before calling the backend", async () => {
    const user = userEvent.setup()
    renderPage()
    await screen.findAllByRole("button", { name: "添加账号" })
    await user.click(screen.getAllByRole("button", { name: "添加账号" })[0])
    await user.type(screen.getByLabelText("账号名称"), "主账号")
    await user.click(screen.getByRole("button", { name: "保存账号" }))
    expect(screen.getByText(/请输入有效的 API Key/)).toBeInTheDocument()
    expect(createAccount).not.toHaveBeenCalled()
  })

  it("adds two different API Keys to the same provider through separate dialogs", async () => {
    const user = userEvent.setup()
    renderPage()
    const addButtons = await screen.findAllByRole("button", { name: "添加账号" })
    const addButton = addButtons[0]

    for (const [name, key] of [
      ["主账号", "first-test-key"],
      ["备用账号", "second-test-key"],
    ]) {
      await user.click(addButton)
      const dialog = screen.getByRole("dialog", { name: "添加 OpenAI 账号" })
      expect(within(dialog).getByLabelText("账号名称")).toHaveFocus()
      await user.type(within(dialog).getByLabelText("账号名称"), name)
      await user.type(within(dialog).getByLabelText("API Key"), key)
      await user.click(within(dialog).getByRole("button", { name: "保存账号" }))
      await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    }

    expect(createAccount).toHaveBeenCalledTimes(2)
    expect(createAccount).toHaveBeenNthCalledWith(1, {
      providerId: "builtin_openai",
      name: "主账号",
      apiKey: "first-test-key",
    })
    expect(createAccount).toHaveBeenNthCalledWith(2, {
      providerId: "builtin_openai",
      name: "备用账号",
      apiKey: "second-test-key",
    })
    expect(screen.getByText("2 个账号")).toBeInTheDocument()
    expect(screen.getByText("主账号")).toBeInTheDocument()
    expect(screen.getByText("备用账号")).toBeInTheDocument()
    expect(screen.queryByDisplayValue("first-test-key")).not.toBeInTheDocument()
    expect(screen.queryByDisplayValue("second-test-key")).not.toBeInTheDocument()
  })

  it("closes the dialog with Escape and returns focus to its add button", async () => {
    const user = userEvent.setup()
    renderPage()
    const addButton = (await screen.findAllByRole("button", { name: "添加账号" }))[0]
    await user.click(addButton)
    expect(screen.getByRole("dialog")).toBeInTheDocument()
    await user.keyboard("{Escape}")
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    expect(addButton).toHaveFocus()
    expect(createAccount).not.toHaveBeenCalled()
  })

  it("keeps the dialog open while an account is being saved", async () => {
    const user = userEvent.setup()
    vi.mocked(createAccount).mockImplementationOnce(() => new Promise(() => {}))
    renderPage()
    await user.click((await screen.findAllByRole("button", { name: "添加账号" }))[0])
    const dialog = screen.getByRole("dialog", { name: "添加 OpenAI 账号" })
    await user.type(within(dialog).getByLabelText("账号名称"), "主账号")
    await user.type(within(dialog).getByLabelText("API Key"), "test-key")
    await user.click(within(dialog).getByRole("button", { name: "保存账号" }))
    expect(within(dialog).getByRole("button", { name: "正在保存…" })).toBeDisabled()
    await user.keyboard("{Escape}")
    expect(screen.getByRole("dialog")).toBeInTheDocument()
    expect(within(dialog).getByRole("button", { name: "取消" })).toBeDisabled()
  })

  it("expands a provider in place without leaving the accounts route", async () => {
    const user = userEvent.setup()
    const { router } = renderPage()
    const builtIn = await screen.findByRole("button", { name: "查看DeepSeek账号详情" })
    expect(screen.getByRole("button", { name: "查看Custom API账号详情" })).toBeInTheDocument()

    await user.click(builtIn)

    // 原位展开：地址仍是概览页，详情直接出现在同一页里。
    expect(router.state.location.pathname).toBe("/accounts")
    const panel = screen.getByRole("region", { name: "DeepSeek" })
    expect(within(panel).getByRole("heading", { name: "DeepSeek 账号" })).toBeInTheDocument()
    // 其他提供商压缩成底部 Dock，被展开的那个不再出现在 Dock 或卡片墙里。
    const dock = screen.getByRole("group", { name: "切换提供商" })
    expect(within(dock).getByRole("button", { name: "查看Custom API账号详情" })).toBeInTheDocument()
    expect(
      within(dock).queryByRole("button", { name: "查看DeepSeek账号详情" }),
    ).not.toBeInTheDocument()
    expect(screen.queryByRole("heading", { name: "Custom API" })).not.toBeInTheDocument()
  })

  it("switches the expanded provider from the bottom dock", async () => {
    const user = userEvent.setup()
    const { router } = renderPage()
    await user.click(await screen.findByRole("button", { name: "查看OpenAI账号详情" }))
    expect(screen.getByRole("region", { name: "OpenAI" })).toBeInTheDocument()

    // Dock 里的 OpenAI 迷你卡片可以随时回到顶部面板。
    await user.click(screen.getByRole("button", { name: "查看DeepSeek账号详情" }))

    expect(screen.getByRole("region", { name: "DeepSeek" })).toBeInTheDocument()
    expect(screen.queryByRole("region", { name: "OpenAI" })).not.toBeInTheDocument()
    expect(
      within(screen.getByRole("group", { name: "切换提供商" })).getByRole("button", {
        name: "查看OpenAI账号详情",
      }),
    ).toBeInTheDocument()
    expect(router.state.location.pathname).toBe("/accounts")
  })

  it("collapses the expanded panel back into the provider grid", async () => {
    const user = userEvent.setup()
    renderPage()
    await user.click(await screen.findByRole("button", { name: "查看OpenAI账号详情" }))
    expect(screen.getByRole("group", { name: "切换提供商" })).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "返回账号" }))

    expect(screen.queryByRole("region", { name: "OpenAI" })).not.toBeInTheDocument()
    expect(screen.queryByRole("group", { name: "切换提供商" })).not.toBeInTheDocument()
    expect(screen.getByRole("heading", { name: "OpenAI" })).toBeInTheDocument()
    expect(screen.getAllByRole("button", { name: "添加账号" })).toHaveLength(10)
  })

  it("keeps provider cards at a fixed height while retaining all account rows", async () => {
    saved = Array.from({ length: 7 }, (_, index) => ({
      id: `acct_${String(index).padStart(32, "0")}`,
      providerId: "builtin_deepseek",
      name: `DeepSeek ${index + 1}`,
      enabled: true,
      createdAt: "2026-09-28T00:00:00Z",
      updatedAt: "2026-09-28T00:00:00Z",
    }))
    renderPage()
    const list = await screen.findByRole("region", { name: "DeepSeek的账号列表" })
    expect(list.closest('[data-slot="card"]')).toHaveClass("h-[320px]")
    expect(
      screen.getByRole("heading", { name: "OpenAI" }).closest('[data-slot="card"]'),
    ).toHaveClass("h-[320px]")
    expect(within(list).getAllByRole("listitem")).toHaveLength(7)
    expect(screen.getByText("7 个账号")).toBeInTheDocument()
  })

  it("keeps focus on the expanded panel and restores it without implicit scrolling", async () => {
    const user = userEvent.setup()
    const focus = vi.spyOn(HTMLElement.prototype, "focus")
    renderPage()
    await user.click(await screen.findByRole("button", { name: "查看OpenAI账号详情" }))
    expect(screen.getByRole("region", { name: "OpenAI" })).toHaveFocus()
    expect(focus).toHaveBeenCalledWith({ preventScroll: true })
    await user.click(screen.getByRole("button", { name: "返回账号" }))
    expect(screen.getByRole("button", { name: "查看OpenAI账号详情" })).toHaveFocus()
    focus.mockRestore()
  })

  it("keeps the route and all other providers available after repeated switches", async () => {
    const user = userEvent.setup()
    const { router } = renderPage()
    for (const name of ["OpenAI", "DeepSeek", "Google", "Custom API", "OpenAI"]) {
      await user.click(await screen.findByRole("button", { name: `查看${name}账号详情` }))
      expect(screen.getByRole("region", { name })).toBeInTheDocument()
      const dock = screen.getByRole("group", { name: "切换提供商" })
      expect(dock.querySelectorAll(".bento-dock-card")).toHaveLength(9)
      expect(within(dock).queryByRole("button", { name: `查看${name}账号详情` })).toBeNull()
      expect(router.state.location.pathname).toBe("/accounts")
    }
  })
})
