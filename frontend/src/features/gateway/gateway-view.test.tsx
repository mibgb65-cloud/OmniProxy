import { act, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { getAppInfo, listAccounts, listProviders } from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import {
  accounts,
  appInfo,
  providers,
  renderGateway,
  resetGatewayRuntime,
} from "./gateway-test-helpers"

vi.mock("../../services/gateway", () => ({
  getGatewayInfo: vi.fn(),
  startGateway: vi.fn(),
  stopGateway: vi.fn(),
}))

vi.mock("../../services/omniproxy", () => ({
  getAppInfo: vi.fn(),
  listProviders: vi.fn(),
  listAccounts: vi.fn(),
}))
vi.mock("@wailsio/runtime", () => ({
  Clipboard: { SetText: vi.fn() },
  Events: { On: vi.fn(() => () => {}) },
}))

describe("网关页面", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    resetGatewayRuntime()
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    vi.mocked(getAppInfo).mockResolvedValue(appInfo)
    vi.mocked(listProviders).mockResolvedValue(providers)
    vi.mocked(listAccounts).mockResolvedValue(accounts)
  })

  it("仅有 Atria API Key 账号时接入检查显示已配置，排除订阅账号", async () => {
    vi.mocked(listProviders).mockResolvedValue([
      { ...providers[0], id: "builtin_atria", kind: "atria", name: "Atria" },
    ])
    vi.mocked(listAccounts).mockResolvedValue([
      { ...accounts[0], providerId: "builtin_atria", authMethod: "api_key" },
      {
        ...accounts[0],
        id: "subscription",
        providerId: "builtin_atria",
        authMethod: "subscription_login",
      },
    ])
    renderGateway("diagnostics")
    await waitFor(() => expect(screen.getAllByText("已配置 · 1 个")).toHaveLength(2))
    expect(screen.queryByText("待配置")).not.toBeInTheDocument()
  })

  it("按选项卡展示实际配置，保留运行控制和元数据检查边界", async () => {
    const user = userEvent.setup()
    renderGateway()
    expect(await screen.findByText("http://127.0.0.1:43210")).toBeInTheDocument()
    expect(screen.getByText("开发环境")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "启动网关" })).toBeEnabled()
    expect(screen.getByRole("link", { name: /模型路由/ })).toHaveAttribute("href", "/routing")
    expect(screen.queryByRole("button", { name: "测试请求" })).not.toBeInTheDocument()
    await user.click(screen.getByRole("tab", { name: "测试诊断" }))
    expect(screen.queryByRole("button", { name: "测试请求" })).not.toBeInTheDocument()
    expect(screen.queryByRole("heading", { name: "请求验证" })).not.toBeInTheDocument()
    expect(screen.queryByRole("checkbox", { name: "流式验证（SSE）" })).not.toBeInTheDocument()
    expect(screen.queryByText("运行中")).not.toBeInTheDocument()
    expect(screen.getByText("检查已保存的配置，不调用模型，也不验证密钥。")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "管理提供商" })).toHaveAttribute("href", "/providers")
    expect(screen.getByRole("link", { name: "管理账号" })).toHaveAttribute("href", "/accounts")
    expect(screen.getByRole("link", { name: /请求记录/ })).toHaveAttribute("href", "/requests")
    expect(screen.getByRole("link", { name: /运行日志/ })).toHaveAttribute("href", "/logs")
    await waitFor(() => expect(screen.getAllByText("已配置 · 1 个")).toHaveLength(2))
  })

  it("加载过程中不将未知资源显示为零或允许复制地址", async () => {
    const user = userEvent.setup()
    vi.mocked(getAppInfo).mockReturnValue(new Promise(() => {}))
    vi.mocked(listProviders).mockReturnValue(new Promise(() => {}))
    vi.mocked(listAccounts).mockReturnValue(new Promise(() => {}))
    renderGateway()
    await screen.findByRole("tab", { name: "运行概览" })
    expect(screen.getAllByText("读取中…").length).toBeGreaterThan(0)
    expect(screen.queryByText("0")).not.toBeInTheDocument()
    await user.click(screen.getByRole("tab", { name: "手动接入" }))
    expect(screen.getByRole("button", { name: "复制配置地址" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "正在刷新…" })).toBeDisabled()
  })

  it("空资源提供配置入口，并允许启动后补充接入配置", async () => {
    const user = userEvent.setup()
    vi.mocked(listProviders).mockResolvedValue([])
    vi.mocked(listAccounts).mockResolvedValue([])
    renderGateway()
    await screen.findByText("http://127.0.0.1:43210")
    expect(screen.getAllByText("0")).toHaveLength(2)
    await user.click(screen.getByRole("tab", { name: "测试诊断" }))
    await waitFor(() => expect(screen.getAllByText("待配置")).toHaveLength(2))
    expect(screen.getByRole("button", { name: "启动网关" })).toBeEnabled()
  })

  it("读取失败不泄露原始错误，刷新后恢复真实配置", async () => {
    const user = userEvent.setup()
    vi.mocked(getAppInfo).mockRejectedValueOnce(new Error("private backend details"))
    renderGateway("manual")
    expect(await screen.findByRole("alert")).toHaveTextContent("无法读取代理配置，请刷新重试。")
    expect(screen.queryByText(/private backend/)).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "复制配置地址" })).toBeDisabled()
    await user.click(screen.getByRole("button", { name: "刷新接入检查" }))
    expect(await screen.findByText("http://127.0.0.1:43210/v1")).toBeInTheDocument()
    expect(getAppInfo).toHaveBeenCalledTimes(2)
    expect(listProviders).toHaveBeenCalledTimes(2)
    expect(listAccounts).toHaveBeenCalledTimes(2)
  })

  it("刷新提供商失败时不继续将旧资源报告为已配置", async () => {
    const user = userEvent.setup()
    renderGateway()
    await screen.findByRole("tab", { name: "测试诊断" })
    await user.click(screen.getByRole("tab", { name: "测试诊断" }))
    await waitFor(() => expect(screen.getAllByText("已配置 · 1 个")).toHaveLength(2))
    vi.mocked(listProviders).mockRejectedValueOnce(new Error("offline"))
    await user.click(screen.getByRole("button", { name: "刷新接入检查" }))
    await waitFor(() => expect(screen.getAllByText("无法检查")).toHaveLength(2))
    expect(screen.queryByText("已配置 · 1 个")).not.toBeInTheDocument()
    expect(screen.getAllByText("读取失败")).toHaveLength(2)
    expect(screen.queryByText("0")).not.toBeInTheDocument()
  })

  it("阻止复制无效端口生成的地址", async () => {
    vi.mocked(getAppInfo).mockResolvedValue({ ...appInfo, proxyPort: 65536 })
    renderGateway("manual")
    expect(await screen.findByRole("alert")).toHaveTextContent("代理端口无效")
    expect(screen.getByRole("button", { name: "复制配置地址" })).toBeDisabled()
    expect(screen.queryByText(/http:\/\/127.0.0.1:/)).not.toBeInTheDocument()
  })

  it("独立显示账号读取失败，不将已读取提供商视为失败", async () => {
    vi.mocked(listAccounts).mockRejectedValueOnce(new Error("account read failed"))
    renderGateway("diagnostics")
    await screen.findByText("已配置 · 1 个")
    const checks = screen.getByRole("heading", { name: "接入检查" }).closest('[data-slot="card"]')!
    expect(within(checks as HTMLElement).getByText("无法检查")).toBeInTheDocument()
    expect(screen.queryByText("0")).not.toBeInTheDocument()
  })

  it("切换语言后显示英文页面和状态", async () => {
    renderGateway("manual")
    await screen.findByText("http://127.0.0.1:43210/v1")
    act(() => useUIStore.setState({ language: "en" }))
    expect(await screen.findByRole("heading", { name: "Gateway" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Start gateway" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "Copy configured address" })).toBeEnabled()
  })
})
