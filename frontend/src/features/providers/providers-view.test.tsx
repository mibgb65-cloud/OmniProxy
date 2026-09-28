import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import {
  createProvider,
  deleteProvider,
  getProviderCatalog,
  listProviders,
  updateProvider,
  type ProviderConfiguration,
  type ProviderDefinition,
} from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import { ProvidersView } from "./providers-view"

vi.mock("../../services/omniproxy", () => ({
  getProviderCatalog: vi.fn(),
  listProviders: vi.fn(),
  createProvider: vi.fn(),
  updateProvider: vi.fn(),
  deleteProvider: vi.fn(),
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
  authMethods: ["api_key"],
  plannedAuthMethods: kind === "openai" ? ["subscription_login"] : null,
}))

let saved: ProviderConfiguration[] = []

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <I18nProvider>
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      </I18nProvider>
    )
  }
  return render(<ProvidersView />, { wrapper: Wrapper })
}

describe("ProvidersView", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    saved = []
    vi.mocked(getProviderCatalog).mockResolvedValue(catalog)
    vi.mocked(listProviders).mockImplementation(async () => saved)
    vi.mocked(createProvider).mockImplementation(async (input) => {
      const item: ProviderConfiguration = {
        id: "prov_0123456789abcdef0123456789abcdef",
        ...input,
        createdAt: "2026-09-28T00:00:00Z",
        updatedAt: "2026-09-28T00:00:00Z",
      }
      saved = [item, ...saved]
      return item
    })
    vi.mocked(updateProvider).mockImplementation(async (id, input) => {
      const item = { ...saved.find((entry) => entry.id === id)!, ...input }
      saved = saved.map((entry) => (entry.id === id ? item : entry))
      return item
    })
    vi.mocked(deleteProvider).mockImplementation(async (id) => {
      saved = saved.filter((entry) => entry.id !== id)
    })
  })

  it("shows all nine presets and marks subscription login as planned", async () => {
    renderPage()
    expect(await screen.findByRole("button", { name: "添加配置：OpenAI" })).toBeInTheDocument()
    expect(screen.getAllByRole("button", { name: /^添加配置：/ })).toHaveLength(9)
    expect(screen.getByText("订阅登录 · 待接入")).toBeInTheDocument()
    expect(screen.getByText("还没有提供商配置。请从上方预设中添加。")).toBeInTheDocument()
  })

  it("validates the endpoint and saves a provider configuration", async () => {
    const user = userEvent.setup()
    renderPage()
    await user.click(await screen.findByRole("button", { name: "添加配置：小米 MiMo" }))
    await user.clear(screen.getByRole("textbox", { name: "配置名称" }))
    await user.type(screen.getByRole("textbox", { name: "配置名称" }), "MiMo Token Plan")
    await user.type(
      screen.getByRole("textbox", { name: "Base URL（可选）" }),
      "http://remote.example/v1",
    )
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    expect(screen.getByText("远程地址必须使用 HTTPS；HTTP 仅允许本机地址。")).toBeInTheDocument()
    expect(createProvider).not.toHaveBeenCalled()

    await user.clear(screen.getByRole("textbox", { name: "Base URL（可选）" }))
    await user.type(
      screen.getByRole("textbox", { name: "Base URL（可选）" }),
      "https://plan.example/v1",
    )
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    await waitFor(() => {
      expect(createProvider).toHaveBeenCalledWith({
        kind: "mimo",
        name: "MiMo Token Plan",
        baseUrl: "https://plan.example/v1",
        enabled: true,
      })
    })
    expect(await screen.findByText("MiMo Token Plan")).toBeInTheDocument()
  })

  it("edits, disables, and deletes a saved configuration", async () => {
    const user = userEvent.setup()
    saved = [
      {
        id: "prov_0123456789abcdef0123456789abcdef",
        kind: "deepseek",
        name: "主配置",
        baseUrl: "",
        enabled: true,
        createdAt: "2026-09-28T00:00:00Z",
        updatedAt: "2026-09-28T00:00:00Z",
      },
    ]
    renderPage()
    await user.click(await screen.findByRole("button", { name: "停用" }))
    expect(await screen.findByText("已停用")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "编辑" }))
    await user.clear(screen.getByRole("textbox", { name: "配置名称" }))
    await user.type(screen.getByRole("textbox", { name: "配置名称" }), "备用配置")
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    expect(await screen.findByText("备用配置")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "删除" }))
    expect(screen.getByText("确定删除“备用配置”吗？")).toBeInTheDocument()
    await user.click(screen.getAllByRole("button", { name: "删除" })[1])
    expect(await screen.findByText("还没有提供商配置。请从上方预设中添加。")).toBeInTheDocument()
  })

  it("explains why a provider with accounts cannot be deleted", async () => {
    const user = userEvent.setup()
    saved = [
      {
        id: "prov_0123456789abcdef0123456789abcdef",
        kind: "openai",
        name: "OpenAI",
        baseUrl: "",
        enabled: true,
        createdAt: "2026-09-28T00:00:00Z",
        updatedAt: "2026-09-28T00:00:00Z",
      },
    ]
    vi.mocked(deleteProvider).mockRejectedValue(new Error("provider: has accounts"))
    renderPage()
    await user.click(await screen.findByRole("button", { name: "删除" }))
    await user.click(screen.getAllByRole("button", { name: "删除" })[1])
    expect(await screen.findByRole("alert")).toHaveTextContent("该配置下仍有账号")
    expect(screen.getByText("OpenAI", { selector: "h3" })).toBeInTheDocument()
  })
})
