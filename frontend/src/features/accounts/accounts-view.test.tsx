import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
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
  createdAt: "2026-09-28T00:00:00Z",
  updatedAt: "2026-09-28T00:00:00Z",
}
let saved: AccountItem[] = []

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <I18nProvider>
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      </I18nProvider>
    )
  }
  return render(<AccountsView />, { wrapper: Wrapper })
}

describe("AccountsView", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    saved = []
    vi.mocked(getProviderCatalog).mockResolvedValue(catalog)
    vi.mocked(listProviders).mockResolvedValue([...providers, custom])
    vi.mocked(listAccounts).mockImplementation(async () => saved)
    vi.mocked(createAccount).mockImplementation(async (input) => {
      const item: AccountItem = {
        id: "acct_0123456789abcdef0123456789abcdef",
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
    vi.spyOn(window, "confirm").mockReturnValue(true)
    renderPage()
    expect((await screen.findAllByRole("button", { name: "添加账号" })).length).toBe(10)
    expect(screen.getByRole("heading", { name: "Custom API" })).toBeInTheDocument()
    await user.click(screen.getAllByRole("button", { name: "添加账号" })[0])
    expect(screen.getByText("添加 OpenAI 账号")).toBeInTheDocument()
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
    expect(within(account.closest("li")!).getByText("API Key · 已启用")).toBeInTheDocument()
    expect(screen.queryByText("test-secret-not-real")).not.toBeInTheDocument()
    expect(screen.queryByDisplayValue("test-secret-not-real")).not.toBeInTheDocument()
    await user.click(screen.getByRole("switch", { name: "切换账号“主账号”的启用状态" }))
    await waitFor(() => expect(setAccountEnabled).toHaveBeenCalledWith(saved[0].id, false))
    await user.click(screen.getByRole("button", { name: "删除账号“主账号”" }))
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
})
