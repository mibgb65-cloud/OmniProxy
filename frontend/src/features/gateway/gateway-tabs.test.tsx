import { act, isInaccessible, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { Events } from "@wailsio/runtime"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { startGateway, type GatewayInfo } from "../../services/gateway"
import { getAppInfo, listAccounts, listProviders } from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import {
  accounts,
  appInfo,
  gatewayInfo,
  listAgentClientsMock as listAgentClients,
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

describe("网关选项卡导航", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    resetGatewayRuntime()
    useUIStore.setState({ language: "zh" })
    vi.mocked(getAppInfo).mockResolvedValue(appInfo)
    vi.mocked(listProviders).mockResolvedValue(providers)
    vi.mocked(listAccounts).mockResolvedValue(accounts)
  })

  it("默认概览仅显示当前面板，访问客户端后才检测，切换不重复订阅", async () => {
    const user = userEvent.setup()
    renderGateway("overview", { entry: "/gateway" })
    expect(await screen.findByRole("tab", { name: "运行概览", selected: true })).toBeInTheDocument()
    expect(screen.getAllByRole("tab")).toHaveLength(4)
    expect(screen.getAllByRole("tabpanel")).toHaveLength(1)
    expect(listAgentClients).not.toHaveBeenCalled()
    await user.click(screen.getByRole("tab", { name: "客户端接入" }))
    await waitFor(() => expect(listAgentClients).toHaveBeenCalledTimes(1))
    await user.click(screen.getByRole("tab", { name: "手动接入" }))
    await user.click(screen.getByRole("tab", { name: "客户端接入" }))
    expect(listAgentClients).toHaveBeenCalledTimes(1)
    expect(Events.On).toHaveBeenCalledTimes(2)
    expect(
      vi.mocked(Events.On).mock.calls.filter((call) => call[0] === "activity:updated"),
    ).toHaveLength(1)
    await user.click(screen.getByRole("button", { name: "刷新接入检查" }))
    await waitFor(() => expect(listAgentClients).toHaveBeenCalledTimes(2))
  })

  it.each(["unknown", "123", "null", "%5B%22manual%22%5D", "x".repeat(300)])(
    "非法参数 %s 回到概览",
    async (tab) => {
      renderGateway("overview", { entry: `/gateway?tab=${tab}` })
      expect(
        await screen.findByRole("tab", { name: "运行概览", selected: true }),
      ).toBeInTheDocument()
      expect(screen.getByRole("heading", { name: "接入与运行配置" })).toBeInTheDocument()
      expect(screen.queryByRole("region", { name: "Codex 接口" })).not.toBeInTheDocument()
    },
  )

  it("直接打开手动接入，前进后退保留模式并隔离隐藏面板焦点", async () => {
    const user = userEvent.setup()
    const { router } = renderGateway("manual")
    const card = within(await screen.findByRole("region", { name: "Codex 接口" }))
    await user.click(card.getByRole("button", { name: "Desktop" }))
    await user.click(screen.getByRole("tab", { name: "测试诊断" }))
    await waitFor(() => expect(router.state.location.search).toEqual({ tab: "diagnostics" }))
    const manualId = screen.getByRole("tab", { name: "手动接入" }).getAttribute("aria-controls")!
    const hiddenManual = document.getElementById(manualId)!
    expect(isInaccessible(hiddenManual)).toBe(true)
    expect(screen.queryByRole("button", { name: "复制 Codex 配置" })).not.toBeInTheDocument()
    await act(async () => router.history.back())
    await screen.findByRole("tab", { name: "手动接入", selected: true })
    expect(card.getByRole("button", { name: "Desktop" })).toHaveAttribute("aria-pressed", "true")
    await act(async () => router.history.forward())
    expect(await screen.findByRole("tab", { name: "测试诊断", selected: true })).toBeInTheDocument()
    expect(screen.getAllByRole("tabpanel")).toHaveLength(1)
  })

  it("方向键移动焦点，Enter 激活页面并保持焦点", async () => {
    const user = userEvent.setup()
    renderGateway()
    const overview = await screen.findByRole("tab", { name: "运行概览" })
    overview.focus()
    await user.keyboard("{ArrowRight}")
    const clients = screen.getByRole("tab", { name: "客户端接入" })
    expect(clients).toHaveFocus()
    expect(overview).toHaveAttribute("aria-selected", "true")
    await user.keyboard("{Enter}")
    expect(await screen.findByRole("tab", { name: "客户端接入", selected: true })).toHaveFocus()
    await user.keyboard("{End}{Enter}")
    expect(await screen.findByRole("tab", { name: "测试诊断", selected: true })).toHaveFocus()
  })

  it("选项卡切换期间公共启停与侧栏共享互斥和操作结果", async () => {
    let finish!: (value: GatewayInfo) => void
    vi.mocked(startGateway).mockReturnValue(
      new Promise((resolve) => {
        finish = resolve
      }),
    )
    const user = userEvent.setup()
    renderGateway("overview", { sidebar: true })
    await waitFor(() => expect(screen.getAllByRole("button", { name: "启动网关" })).toHaveLength(2))
    await user.click(screen.getAllByRole("button", { name: "启动网关" })[1]!)
    await user.click(screen.getByRole("tab", { name: "手动接入" }))
    for (const button of screen.getAllByRole("button", { name: "正在启动…" }))
      expect(button).toBeDisabled()
    expect(startGateway).toHaveBeenCalledTimes(1)
    await act(async () =>
      finish({ ...gatewayInfo, runtime: { ...gatewayInfo.runtime, state: "running" } }),
    )
    await waitFor(() => expect(screen.getAllByRole("button", { name: "停止网关" })).toHaveLength(2))
    for (const button of screen.getAllByRole("button", { name: "停止网关" }))
      expect(button).toBeEnabled()
  })
})
