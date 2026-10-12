import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import { act, isInaccessible, render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { getGatewayInfo, startGateway, stopGateway, type GatewayInfo } from "../../services/gateway"
import { listAccounts, type AccountItem } from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import { SidebarGatewayStatus } from "./sidebar-gateway-status"

vi.mock("../../services/gateway", () => ({
  getGatewayInfo: vi.fn(),
  startGateway: vi.fn(),
  stopGateway: vi.fn(),
}))
vi.mock("../../services/omniproxy", () => ({ listAccounts: vi.fn() }))
vi.mock("@wailsio/runtime", () => ({ Events: { On: vi.fn(() => vi.fn()) } }))

const stopped: GatewayInfo = {
  runtime: { state: "stopped", address: "", port: 43210, startedAt: "", errorCode: "" },
  models: [],
  protocol: "Chat Completions",
  requestTimeoutSeconds: 120,
}
const running: GatewayInfo = { ...stopped, runtime: { ...stopped.runtime, state: "running" } }
const accounts: AccountItem[] = [true, false, true].map((enabled, index) => ({
  id: String(index),
  providerId: "provider",
  name: "Account",
  enabled,
  createdAt: "",
  updatedAt: "",
}))

function renderStatus(collapsed = false, duplicate = false) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <I18nProvider>
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      </I18nProvider>
    )
  }
  const result = render(
    <>
      <SidebarGatewayStatus collapsed={collapsed} />
      {duplicate ? <SidebarGatewayStatus collapsed={!collapsed} /> : null}
    </>,
    { wrapper: Wrapper },
  )
  return { ...result, client }
}

describe("侧栏网关状态", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    vi.mocked(getGatewayInfo).mockResolvedValue(stopped)
    vi.mocked(startGateway).mockResolvedValue(running)
    vi.mocked(stopGateway).mockResolvedValue(stopped)
    vi.mocked(listAccounts).mockResolvedValue(accounts)
  })

  it("显示实际端口及全部已添加账号，缓存失效后更新数量", async () => {
    const { client } = renderStatus()
    expect(await screen.findByText("43210")).toBeInTheDocument()
    expect(screen.getByText("3")).toBeInTheDocument()
    expect(screen.getByRole("status")).toHaveTextContent("网关：已停止")
    vi.mocked(listAccounts).mockResolvedValue([])
    await act(async () => {
      await client.invalidateQueries({ queryKey: ["accounts"] })
    })
    expect(await screen.findByText("0")).toBeInTheDocument()
  })

  it("通过电源按钮启动及停止真实网关接口", async () => {
    const user = userEvent.setup()
    renderStatus()
    await screen.findByText("已停止")
    await user.click(screen.getByRole("button", { name: "启动网关" }))
    expect(await screen.findByText("运行中")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "停止网关" }))
    expect(await screen.findByText("已停止")).toBeInTheDocument()
    expect(startGateway).toHaveBeenCalledTimes(1)
    expect(stopGateway).toHaveBeenCalledTimes(1)
  })

  it("操作未完成时同步禁用其他入口并显示正在启动", async () => {
    let finish!: (value: GatewayInfo) => void
    vi.mocked(startGateway).mockReturnValue(
      new Promise((resolve) => {
        finish = resolve
      }),
    )
    const user = userEvent.setup()
    renderStatus(false, true)
    await waitFor(() => expect(screen.getAllByRole("button", { name: "启动网关" })).toHaveLength(2))
    await user.click(screen.getAllByRole("button", { name: "启动网关" })[0]!)
    await waitFor(() => {
      for (const button of screen.getAllByRole("button", { name: "正在启动…" })) {
        expect(button).toBeDisabled()
        expect(button).toHaveAttribute("aria-busy", "true")
      }
    })
    await act(async () => {
      finish(running)
    })
    await waitFor(() => expect(screen.getAllByRole("button", { name: "停止网关" })).toHaveLength(2))
    expect(startGateway).toHaveBeenCalledTimes(1)
  })

  it("读取失败时不误报停止或零账号，不暴露原始错误", async () => {
    vi.mocked(getGatewayInfo).mockRejectedValue(new Error("private runtime detail"))
    vi.mocked(listAccounts).mockRejectedValue(new Error("private account detail"))
    renderStatus()
    expect(await screen.findByRole("alert")).toHaveTextContent("无法读取网关状态")
    expect(screen.getByRole("status")).toHaveTextContent("状态不可用")
    expect(screen.getByRole("button", { name: "启动网关" })).toBeDisabled()
    expect(screen.queryByText("已停止")).not.toBeInTheDocument()
    expect(screen.queryByText("0")).not.toBeInTheDocument()
    expect(screen.queryByText(/private/)).not.toBeInTheDocument()
  })

  it("账号读取失败不影响网关控制", async () => {
    vi.mocked(listAccounts).mockRejectedValue(new Error("private account detail"))
    renderStatus()
    expect(await screen.findByRole("alert")).toHaveTextContent("无法读取账号数量")
    expect(screen.getByRole("button", { name: "启动网关" })).toBeEnabled()
    expect(screen.queryByText("0")).not.toBeInTheDocument()
  })

  it.each([0, -1, 65536, 1.5])("无效端口 %s 不显示为可用端口", async (port) => {
    vi.mocked(getGatewayInfo).mockResolvedValue({
      ...stopped,
      runtime: { ...stopped.runtime, port },
    })
    renderStatus()
    expect(await screen.findByText("已停止")).toBeInTheDocument()
    expect(screen.getByText("读取失败")).toBeInTheDocument()
  })

  it("启动失败显示安全错误，并可重试", async () => {
    vi.mocked(startGateway).mockRejectedValueOnce(
      new Error("gateway: port_unavailable private detail"),
    )
    const user = userEvent.setup()
    renderStatus()
    await screen.findByText("已停止")
    await user.click(screen.getByRole("button", { name: "启动网关" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("代理端口不可用")
    expect(screen.queryByText(/private/)).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "启动网关" }))
    expect(await screen.findByText("运行中")).toBeInTheDocument()
    expect(screen.queryByRole("alert")).not.toBeInTheDocument()
  })

  it("其他入口重试成功后同步清理过期错误", async () => {
    vi.mocked(startGateway).mockRejectedValueOnce(new Error("gateway: port_unavailable"))
    const user = userEvent.setup()
    renderStatus(false, true)
    await waitFor(() => expect(screen.getAllByRole("button", { name: "启动网关" })).toHaveLength(2))
    await user.click(screen.getAllByRole("button", { name: "启动网关" })[0]!)
    await waitFor(() => expect(screen.getAllByRole("alert")).toHaveLength(2))
    await user.click(screen.getAllByRole("button", { name: "启动网关" })[1]!)
    await waitFor(() => expect(screen.getAllByRole("button", { name: "停止网关" })).toHaveLength(2))
    expect(screen.queryByRole("alert")).not.toBeInTheDocument()
  })

  it("收起后保留可访问的按钮与完整概况提示", async () => {
    const user = userEvent.setup()
    renderStatus(true)
    await screen.findByText("已停止")
    const button = screen.getByRole("button", { name: "启动网关" })
    await user.hover(button.parentElement!)
    const tooltip = await screen.findByRole("tooltip")
    expect(tooltip).toHaveTextContent("端口：43210")
    expect(tooltip).toHaveTextContent("账号数量：3")
    expect(
      isInaccessible(within(screen.getByRole("region", { name: "网关" })).getByText("端口")),
    ).toBe(true)
    await user.click(button)
    expect(await screen.findByRole("button", { name: "停止网关" })).toBeEnabled()
  })

  it("连续收展保留电源按钮焦点和状态描述，恢复详情可访问性", async () => {
    const { rerender } = renderStatus()
    await screen.findByText("已停止")
    const button = screen.getByRole("button", { name: "启动网关" })
    button.focus()
    for (const collapsed of [true, false, true, false]) {
      rerender(<SidebarGatewayStatus collapsed={collapsed} />)
      expect(screen.getByRole("button", { name: "启动网关" })).toBe(button)
      expect(button).toHaveFocus()
      expect(button).toHaveAccessibleDescription("网关：已停止")
      expect(isInaccessible(screen.getByText("端口"))).toBe(collapsed)
    }
    expect(startGateway).not.toHaveBeenCalled()
    expect(stopGateway).not.toHaveBeenCalled()
  })

  it("响应外部状态事件并在卸载时取消订阅", async () => {
    const unsubscribe = vi.fn()
    vi.mocked(Events.On).mockReturnValue(unsubscribe)
    const { unmount } = renderStatus()
    await screen.findByText("已停止")
    vi.mocked(getGatewayInfo).mockResolvedValue(running)
    const listener = vi
      .mocked(Events.On)
      .mock.calls.find(([name]) => name === "gateway:status")?.[1]
    await act(async () => {
      listener?.({ name: "gateway:status", data: {}, sender: "" })
    })
    expect(await screen.findByText("运行中")).toBeInTheDocument()
    unmount()
    expect(unsubscribe).toHaveBeenCalledTimes(2)
  })
})
