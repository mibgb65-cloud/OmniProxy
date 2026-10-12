import { Events } from "@wailsio/runtime"
import { act, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { getGatewayInfo, startGateway, stopGateway } from "../../services/gateway"
import { getAppInfo, listAccounts, listProviders } from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import {
  accounts,
  appInfo,
  gatewayInfo,
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
  Events: { On: vi.fn(() => vi.fn()) },
}))

describe("网关运行控制", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    resetGatewayRuntime()
    vi.mocked(getAppInfo).mockResolvedValue(appInfo)
    vi.mocked(listProviders).mockResolvedValue(providers)
    vi.mocked(listAccounts).mockResolvedValue(accounts)
  })

  it("启动和停止后展示真实运行状态", async () => {
    const user = userEvent.setup()
    renderGateway("diagnostics")
    await screen.findByText("已停止")
    await user.click(screen.getByRole("button", { name: "启动网关" }))
    expect(await screen.findByText("运行中")).toBeInTheDocument()
    expect(startGateway).toHaveBeenCalledTimes(1)
    await user.click(screen.getByRole("button", { name: "停止网关" }))
    expect(await screen.findByText("已停止")).toBeInTheDocument()
    expect(stopGateway).toHaveBeenCalledTimes(1)
  })

  it("绑定失败显示安全的端口错误，允许重试", async () => {
    const user = userEvent.setup()
    vi.mocked(startGateway).mockRejectedValueOnce(
      new Error("gateway: port_unavailable private detail"),
    )
    renderGateway()
    await screen.findByText("已停止")
    await user.click(screen.getByRole("button", { name: "启动网关" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("代理端口不可用")
    expect(screen.queryByText(/private detail/)).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "启动网关" }))
    expect(await screen.findByText("运行中")).toBeInTheDocument()
  })

  it("状态事件刷新快照，卸载时清理订阅", async () => {
    const unsubscribe = vi.fn()
    vi.mocked(Events.On).mockReturnValueOnce(unsubscribe)
    const { unmount } = renderGateway()
    await screen.findByText("已停止")
    const callback = vi.mocked(Events.On).mock.calls[0]?.[1]
    expect(callback).toBeDefined()
    vi.mocked(getGatewayInfo).mockResolvedValue({
      ...gatewayInfo,
      runtime: { ...gatewayInfo.runtime, state: "running" },
    })
    await act(async () =>
      callback?.({ name: "gateway:status", data: gatewayInfo.runtime, sender: "" }),
    )
    expect(await screen.findByText("运行中")).toBeInTheDocument()
    unmount()
    expect(unsubscribe).toHaveBeenCalledTimes(1)
  })

  it("状态读取失败时不误报为已停止，保留安全错误反馈", async () => {
    vi.mocked(getGatewayInfo).mockRejectedValueOnce(new Error("private runtime error"))
    renderGateway()
    expect(await screen.findByRole("alert")).toHaveTextContent("无法读取网关状态")
    expect(screen.queryByText("已停止")).not.toBeInTheDocument()
    expect(screen.queryByText(/private runtime error/)).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "启动网关" })).toBeDisabled()
  })
})
