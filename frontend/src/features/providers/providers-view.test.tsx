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
  probeProvider,
  setProviderEnabled,
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
  setProviderEnabled: vi.fn(),
  probeProvider: vi.fn(),
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
  defaultBaseUrl: "https://api.example.test/v1",
  models: kind === "openai" ? ["GPT-5.4", "GPT-4.1", "o3"] : ["Model A", "Model B"],
  authMethods: ["api_key"],
  plannedAuthMethods: kind === "openai" ? ["subscription_login"] : null,
}))
const builtins: ProviderConfiguration[] = kinds.map((kind) => ({
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
  name: "My endpoint",
  baseUrl: "https://api.example.test/v1",
  enabled: true,
  createdAt: "2026-09-28T00:00:00Z",
  updatedAt: "2026-09-28T00:00:00Z",
}
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
    saved = builtins.map((item) => ({ ...item }))
    vi.mocked(getProviderCatalog).mockResolvedValue(catalog)
    vi.mocked(listProviders).mockImplementation(async () => saved)
    vi.mocked(createProvider).mockImplementation(async (input) => {
      const item = { ...custom, ...input }
      saved = [item, ...saved]
      return item
    })
    vi.mocked(updateProvider).mockImplementation(async (id, input) => {
      const item = { ...saved.find((entry) => entry.id === id)!, ...input }
      saved = saved.map((entry) => (entry.id === id ? item : entry))
      return item
    })
    vi.mocked(setProviderEnabled).mockImplementation(async (id, enabled) => {
      const item = { ...saved.find((entry) => entry.id === id)!, enabled }
      saved = saved.map((entry) => (entry.id === id ? item : entry))
      return item
    })
    vi.mocked(deleteProvider).mockImplementation(async (id) => {
      saved = saved.filter((entry) => entry.id !== id)
    })
    vi.mocked(probeProvider).mockResolvedValue({ reachable: true, latencyMs: 42, statusCode: 401 })
  })

  it("shows nine enabled built-ins without an add action", async () => {
    renderPage()
    expect(
      await screen.findByRole("button", { name: "切换OpenAI提供商的启用状态" }),
    ).toHaveAttribute("aria-pressed", "true")
    expect(screen.getByRole("heading", { name: "提供商" })).toBeInTheDocument()
    expect(screen.getByRole("heading", { name: "内置提供商" })).toBeInTheDocument()
    expect(screen.getAllByRole("button", { name: /切换.*提供商的启用状态/ })).toHaveLength(9)
    expect(screen.getByText("订阅登录 · 待接入")).toBeInTheDocument()
    expect(screen.getAllByRole("button", { name: "新增自定义提供商" })).toHaveLength(1)
    expect(screen.getByText("还没有自定义提供商。")).toBeInTheDocument()
    expect(screen.queryByText(/九家内置提供商开箱即用/)).not.toBeInTheDocument()
    expect(screen.queryByText(/内置提供商默认启用/)).not.toBeInTheDocument()
    expect(screen.queryByText(/模型为官方预设示例/)).not.toBeInTheDocument()
    expect(screen.queryByText(/九家提供商已预设并默认启用/)).not.toBeInTheDocument()
  })

  it("persists a built-in power toggle without changing the preset", async () => {
    const user = userEvent.setup()
    renderPage()
    await user.click(await screen.findByRole("button", { name: "切换OpenAI提供商的启用状态" }))
    await waitFor(() => expect(setProviderEnabled).toHaveBeenCalledWith("builtin_openai", false))
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "切换OpenAI提供商的启用状态" })).toHaveAttribute(
        "aria-pressed",
        "false",
      ),
    )
    await user.click(screen.getByRole("button", { name: "切换OpenAI提供商的启用状态" }))
    await waitFor(() => expect(setProviderEnabled).toHaveBeenLastCalledWith("builtin_openai", true))
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "切换OpenAI提供商的启用状态" })).toHaveAttribute(
        "aria-pressed",
        "true",
      ),
    )
    expect(createProvider).not.toHaveBeenCalled()
  })

  it("expands preset models and reports unauthenticated endpoint latency", async () => {
    const user = userEvent.setup()
    renderPage()
    await user.click((await screen.findAllByRole("button", { name: /查看全部预设模型/ }))[0])
    expect(screen.getByText("o3")).toBeInTheDocument()
    await user.click(screen.getAllByRole("button", { name: "测试" })[0])
    await waitFor(() => expect(probeProvider).toHaveBeenCalledWith("builtin_openai"))
    const tip = await screen.findByText("地址可达 · 42 ms · HTTP 401")
    expect(tip).toHaveAttribute("data-state", "open")
    expect(screen.getAllByRole("button", { name: "测试" })[0].previousElementSibling).toBe(tip)
    await waitFor(() => expect(tip).toHaveAttribute("data-state", "closed"), { timeout: 5_000 })
    expect(tip).toHaveAttribute("aria-hidden", "true")
  })

  it("blocks repeated power toggles while a provider update is pending", async () => {
    const user = userEvent.setup()
    let finishUpdate!: (item: ProviderConfiguration) => void
    vi.mocked(setProviderEnabled).mockImplementation(
      () => new Promise((resolve) => (finishUpdate = resolve)),
    )
    renderPage()
    const button = await screen.findByRole("button", { name: "切换OpenAI提供商的启用状态" })
    await user.click(button)
    await waitFor(() => expect(button).toBeDisabled())
    await user.click(button)
    expect(setProviderEnabled).toHaveBeenCalledTimes(1)
    saved = saved.map((item) => (item.id === "builtin_openai" ? { ...item, enabled: false } : item))
    finishUpdate(saved.find((item) => item.id === "builtin_openai")!)
    await waitFor(() => expect(button).toBeEnabled())
    expect(button).toHaveAttribute("aria-pressed", "false")
  })

  it("requires a URL and creates an enabled custom provider", async () => {
    const user = userEvent.setup()
    renderPage()
    await user.click(await screen.findByRole("button", { name: "新增自定义提供商" }))
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    expect(screen.getByText("请输入 Base URL。")).toBeInTheDocument()
    await user.type(screen.getByRole("textbox", { name: "配置名称" }), "My endpoint")
    await user.type(
      screen.getByRole("textbox", { name: "Base URL" }),
      "https://api.example.test/v1",
    )
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    await waitFor(() =>
      expect(createProvider).toHaveBeenCalledWith({
        name: "My endpoint",
        baseUrl: "https://api.example.test/v1",
      }),
    )
    expect(
      await screen.findByRole("button", { name: "切换My endpoint提供商的启用状态" }),
    ).toHaveAttribute("aria-pressed", "true")
    expect(screen.getAllByRole("button", { name: /切换.*提供商的启用状态/ })).toHaveLength(10)
  })

  it("edits, disables and deletes only the custom provider", async () => {
    const user = userEvent.setup()
    saved = [custom, ...saved]
    renderPage()
    await user.click(await screen.findByRole("button", { name: "编辑" }))
    await user.clear(screen.getByRole("textbox", { name: "配置名称" }))
    await user.type(screen.getByRole("textbox", { name: "配置名称" }), "Backup")
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    expect(await screen.findByText("Backup")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "切换Backup提供商的启用状态" }))
    await waitFor(() => expect(setProviderEnabled).toHaveBeenCalledWith(custom.id, false))
    await user.click(screen.getByRole("button", { name: "删除" }))
    await user.click(screen.getAllByRole("button", { name: "删除" })[1])
    expect(await screen.findByText("还没有自定义提供商。")).toBeInTheDocument()
    expect(screen.getAllByRole("button", { name: /切换.*提供商的启用状态/ })).toHaveLength(9)
  })

  it("keeps an existing provider configuration with an empty URL editable", async () => {
    const user = userEvent.setup()
    saved = [{ ...custom, kind: "mimo", name: "Old MiMo", baseUrl: "" }, ...saved]
    renderPage()
    await user.click(await screen.findByRole("button", { name: "编辑" }))
    expect(screen.getByRole("textbox", { name: "Base URL（可选）" })).toHaveValue("")
    await user.clear(screen.getByRole("textbox", { name: "配置名称" }))
    await user.type(screen.getByRole("textbox", { name: "配置名称" }), "Renamed MiMo")
    await user.click(screen.getByRole("button", { name: "保存配置" }))
    await waitFor(() =>
      expect(updateProvider).toHaveBeenCalledWith(custom.id, { name: "Renamed MiMo", baseUrl: "" }),
    )
  })

  it("explains why a custom provider with accounts cannot be deleted", async () => {
    const user = userEvent.setup()
    saved = [custom, ...saved]
    vi.mocked(deleteProvider).mockRejectedValue(new Error("provider: has accounts"))
    renderPage()
    await user.click(await screen.findByRole("button", { name: "删除" }))
    await user.click(screen.getAllByRole("button", { name: "删除" })[1])
    expect(await screen.findByRole("alert")).toHaveTextContent("该配置下仍有账号")
  })
})
