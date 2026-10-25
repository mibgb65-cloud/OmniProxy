import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import {
  discoverProxyPorts,
  getProviderCatalog,
  getProxySettings,
  listProviders,
  saveProxySettings,
  setProviderProxyMode,
  testProxySettings,
  type ProviderConfiguration,
  type ProxySettings,
} from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import { ProxySettingsDialog } from "./proxy-settings-dialog"

vi.mock("../../services/omniproxy", () => ({
  getProviderCatalog: vi.fn(),
  getProxySettings: vi.fn(),
  saveProxySettings: vi.fn(),
  testProxySettings: vi.fn(),
  discoverProxyPorts: vi.fn(),
  setProviderProxyMode: vi.fn(),
  listProviders: vi.fn(),
  createProvider: vi.fn(),
  updateProvider: vi.fn(),
  deleteProvider: vi.fn(),
  setProviderEnabled: vi.fn(),
  probeProvider: vi.fn(),
}))

const settings: ProxySettings = {
  enabled: true,
  scheme: "auto",
  host: "127.0.0.1",
  port: 10808,
  username: "",
  hasPassword: true,
  noProxy: "localhost, 127.0.0.1",
  detectedScheme: "socks5",
  updatedAt: "2026-10-01T00:00:00Z",
  endpoint: "socks5://127.0.0.1:10808",
}

const providers: ProviderConfiguration[] = [
  {
    id: "builtin_openai",
    kind: "openai",
    source: "builtin",
    name: "openai",
    baseUrl: "",
    enabled: true,
    proxyMode: "inherit",
    createdAt: "2026-09-28T00:00:00Z",
    updatedAt: "2026-09-28T00:00:00Z",
  },
  {
    id: "builtin_deepseek",
    kind: "deepseek",
    source: "builtin",
    name: "deepseek",
    baseUrl: "",
    enabled: true,
    proxyMode: "proxy",
    createdAt: "2026-09-28T00:00:00Z",
    updatedAt: "2026-09-28T00:00:00Z",
  },
  {
    id: "prov_0123456789abcdef0123456789abcdef",
    kind: "openai-compatible",
    source: "custom",
    name: "My endpoint",
    baseUrl: "https://api.example.test/v1",
    enabled: true,
    proxyMode: "direct",
    createdAt: "2026-09-28T00:00:00Z",
    updatedAt: "2026-09-28T00:00:00Z",
  },
]

function renderDialog() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <I18nProvider>
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      </I18nProvider>
    )
  }
  const onClose = vi.fn()
  render(<ProxySettingsDialog onClose={onClose} />, { wrapper: Wrapper })
  return { onClose }
}

/** 设置读取是异步的：等表单落地再断言，避免只看到加载态。 */
async function openForm() {
  const dialog = await screen.findByRole("dialog")
  await within(dialog).findByLabelText("服务器地址")
  return dialog
}

describe("ProxySettingsDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    vi.mocked(getProxySettings).mockResolvedValue(settings)
    vi.mocked(listProviders).mockResolvedValue(providers)
    vi.mocked(getProviderCatalog).mockResolvedValue([])
    vi.mocked(setProviderProxyMode).mockImplementation(async (id, mode) => ({
      ...providers.find((item) => item.id === id)!,
      proxyMode: mode,
    }))
    vi.mocked(saveProxySettings).mockResolvedValue(settings)
    vi.mocked(testProxySettings).mockResolvedValue({
      reachable: true,
      socks5: true,
      http: false,
      auth: false,
      scheme: "socks5",
      errorCode: "",
      endpoint: "127.0.0.1:10808",
      latencyMs: 12,
      detectedAt: "2026-10-01T00:00:00Z",
    })
    vi.mocked(discoverProxyPorts).mockResolvedValue([])
  })

  it("renders the saved configuration as a modal dialog", async () => {
    const { onClose } = renderDialog()
    const dialog = await openForm()
    expect(dialog).toHaveAttribute("aria-modal", "true")
    expect(within(dialog).getByLabelText("服务器地址")).toHaveValue("127.0.0.1")
    expect(within(dialog).getByLabelText("端口")).toHaveValue(10808)
    // 已保存密码时不回显明文，只提示留空即不修改。
    expect(within(dialog).getByLabelText("密码")).toHaveValue("")
    expect(within(dialog).getByPlaceholderText("已保存密码，留空表示不修改")).toBeInTheDocument()
    await userEvent.setup().keyboard("{Escape}")
    await waitFor(() => expect(onClose).toHaveBeenCalled())
  })

  it("shows the specific error when the proxy port does not answer", async () => {
    const user = userEvent.setup()
    vi.mocked(testProxySettings).mockResolvedValue({
      reachable: false,
      socks5: false,
      http: false,
      auth: false,
      scheme: "",
      errorCode: "proxy_unavailable",
      endpoint: "127.0.0.1:10808",
      latencyMs: 0,
      detectedAt: "2026-10-01T00:00:00Z",
    })
    renderDialog()
    await user.click(await screen.findByRole("button", { name: "测试" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("代理端口无响应")
    expect(testProxySettings).toHaveBeenCalledWith({
      enabled: true,
      scheme: "auto",
      host: "127.0.0.1",
      port: 10808,
      username: "",
      password: "",
      clearPassword: false,
      noProxy: "localhost, 127.0.0.1",
    })
  })

  it("explains a protocol mismatch and reports a reachable proxy", async () => {
    const user = userEvent.setup()
    renderDialog()
    await openForm()
    expect(screen.queryByRole("alert")).not.toBeInTheDocument()
    vi.mocked(testProxySettings).mockResolvedValueOnce({
      reachable: false,
      socks5: false,
      http: false,
      auth: false,
      scheme: "",
      errorCode: "proxy_protocol_mismatch",
      endpoint: "127.0.0.1:10808",
      latencyMs: 0,
      detectedAt: "2026-10-01T00:00:00Z",
    })
    await user.click(screen.getByRole("button", { name: "测试" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("端口不是 HTTP/SOCKS 代理")
    await user.click(screen.getByRole("button", { name: "测试" }))
    expect(await screen.findByRole("status")).toHaveTextContent("代理可用 · SOCKS5 · 12 ms")
  })

  it("switches a single provider to the proxy mode", async () => {
    const user = userEvent.setup()
    renderDialog()
    const group = await screen.findByRole("group", { name: "OpenAI的出站方式" })
    await user.click(within(group).getByRole("button", { name: "走代理" }))
    await waitFor(() =>
      expect(setProviderProxyMode).toHaveBeenCalledWith("builtin_openai", "proxy"),
    )
  })

  it("applies the proxy mode to every listed provider in bulk", async () => {
    const user = userEvent.setup()
    renderDialog()
    await openForm()
    await user.click(screen.getByRole("button", { name: "全部走代理" }))
    // 已经走代理的提供商不再重复提交。
    await waitFor(() => expect(setProviderProxyMode).toHaveBeenCalledTimes(2))
    expect(setProviderProxyMode).toHaveBeenCalledWith("builtin_openai", "proxy")
    expect(setProviderProxyMode).toHaveBeenCalledWith(providers[2].id, "proxy")
  })

  it("fills host, port and scheme from a detected local proxy", async () => {
    const user = userEvent.setup()
    vi.mocked(discoverProxyPorts).mockResolvedValue([
      {
        endpoint: "127.0.0.1:7890",
        scheme: "socks5",
        socks5: true,
        http: false,
        auth: false,
        port: 7890,
        latencyMs: 3,
      },
    ])
    renderDialog()
    await openForm()
    await user.click(screen.getByRole("button", { name: "探测本机代理" }))
    await user.click(await screen.findByRole("button", { name: "127.0.0.1:7890 · SOCKS5" }))
    expect(screen.getByLabelText("服务器地址")).toHaveValue("127.0.0.1")
    expect(screen.getByLabelText("端口")).toHaveValue(7890)
    expect(screen.getByLabelText("协议")).toHaveTextContent("SOCKS5")
  })

  it("explains an empty detection result", async () => {
    const user = userEvent.setup()
    renderDialog()
    await openForm()
    await user.click(screen.getByRole("button", { name: "探测本机代理" }))
    expect(
      await screen.findByText("未发现本机代理端口，请手动填写地址和端口。"),
    ).toBeInTheDocument()
  })

  it("saves without resending the stored password", async () => {
    const user = userEvent.setup()
    const { onClose } = renderDialog()
    await openForm()
    await user.click(screen.getByRole("button", { name: "保存" }))
    await waitFor(() => expect(saveProxySettings).toHaveBeenCalled())
    expect(vi.mocked(saveProxySettings).mock.calls[0][0]).toMatchObject({
      password: "",
      clearPassword: false,
    })
    await waitFor(() => expect(onClose).toHaveBeenCalled())
  })

  it("keeps the dialog open and reports a sanitized failure", async () => {
    const user = userEvent.setup()
    const { onClose } = renderDialog()
    await openForm()
    vi.mocked(saveProxySettings).mockRejectedValue(new Error("save failed for 127.0.0.1"))
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(await screen.findByText("保存失败，请检查填写内容后重试。")).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })
})
