import { Clipboard } from "@wailsio/runtime"
import { act, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { getGatewayInfo } from "../../services/gateway"
import { getAppInfo, listAccounts, listProviders } from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import { getGatewayClientConfig } from "./gateway-client-configuration"
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
  Events: { On: vi.fn(() => () => {}) },
}))

describe("Codex 与 Claude 接入卡片", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    resetGatewayRuntime()
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    vi.mocked(getAppInfo).mockResolvedValue(appInfo)
    vi.mocked(listProviders).mockResolvedValue(providers)
    vi.mocked(listAccounts).mockResolvedValue(accounts)
    vi.mocked(Clipboard.SetText).mockResolvedValue()
  })

  it("呈现两个客户端的地址与协议，并移除旧页面副标题", async () => {
    renderGateway("manual")
    await screen.findByText("http://127.0.0.1:43210/v1")
    const codex = within(screen.getByRole("region", { name: "Codex 接口" }))
    const claude = within(screen.getByRole("region", { name: "Claude 接口" }))
    expect(codex.getByRole("textbox", { name: "Base URL" })).toHaveValue(
      "http://127.0.0.1:43210/codex/v1",
    )
    expect(claude.getByRole("textbox", { name: "Base URL" })).toHaveValue(
      "http://127.0.0.1:43210/claude",
    )
    expect(codex.getByText("POST http://127.0.0.1:43210/codex/v1/responses")).toBeInTheDocument()
    expect(claude.getByText("POST http://127.0.0.1:43210/claude/v1/messages")).toBeInTheDocument()
    expect(screen.getAllByText("协议已接入")).toHaveLength(2)
    expect(
      screen.queryByText("运行本地 DeepSeek 代理，检查接入并验证实际请求。"),
    ).not.toBeInTheDocument()
  })

  it.each([
    ["Codex", "http://127.0.0.1:43210/codex/v1"],
    ["Claude", "http://127.0.0.1:43210/claude"],
  ])("复制 %s 的正确 Base URL", async (client, address) => {
    const user = userEvent.setup()
    renderGateway("manual")
    await screen.findByText("http://127.0.0.1:43210/v1")
    const card = within(screen.getByRole("region", { name: `${client} 接口` }))
    await user.click(card.getByRole("button", { name: `复制 ${client} 地址` }))
    await waitFor(() => expect(Clipboard.SetText).toHaveBeenCalledWith(address))
    expect(card.getByRole("status")).toHaveTextContent("已复制")
  })

  it("复制 Codex 预览配置并切换桌面说明", async () => {
    const user = userEvent.setup()
    renderGateway("manual")
    await screen.findByText("http://127.0.0.1:43210/v1")
    const card = within(screen.getByRole("region", { name: "Codex 接口" }))
    await user.click(card.getByRole("button", { name: "复制 Codex 配置" }))
    expect(Clipboard.SetText).toHaveBeenCalledWith(
      getGatewayClientConfig("codex", "http://127.0.0.1:43210/codex/v1"),
    )
    await user.click(card.getByRole("button", { name: "Desktop" }))
    expect(card.getByRole("button", { name: "Desktop" })).toHaveAttribute("aria-pressed", "true")
    expect(card.getByText(/桌面应用读取同一份用户 config.toml/)).toBeInTheDocument()
  })

  it("Claude 桌面模式展示专用配置入口，返回 CLI 后仍能复制 JSON", async () => {
    const user = userEvent.setup()
    renderGateway("manual")
    await screen.findByText("http://127.0.0.1:43210/v1")
    const card = within(screen.getByRole("region", { name: "Claude 接口" }))
    await user.click(card.getByRole("button", { name: "Desktop" }))
    expect(card.getByText(/Configure Third-Party Inference/)).toBeInTheDocument()
    expect(card.queryByText("~/.claude/settings.json")).not.toBeInTheDocument()
    expect(card.queryByRole("button", { name: "复制 Claude 配置" })).not.toBeInTheDocument()
    await user.click(card.getByRole("button", { name: "CLI" }))
    await user.click(card.getByRole("button", { name: "复制 Claude 配置" }))
    expect(Clipboard.SetText).toHaveBeenCalledWith(
      getGatewayClientConfig("claude", "http://127.0.0.1:43210/claude"),
    )
  })

  it("复制失败显示安全反馈，并可再次复制", async () => {
    const user = userEvent.setup()
    vi.mocked(Clipboard.SetText).mockRejectedValueOnce(new Error("private clipboard failure"))
    renderGateway("manual")
    await screen.findByText("http://127.0.0.1:43210/v1")
    const card = within(screen.getByRole("region", { name: "Claude 接口" }))
    await user.click(card.getByRole("button", { name: "复制 Claude 地址" }))
    expect(card.getByRole("alert")).toHaveTextContent("复制失败")
    expect(screen.queryByText(/private clipboard/)).not.toBeInTheDocument()
    await user.click(card.getByRole("button", { name: "复制 Claude 地址" }))
    expect(card.getByRole("status")).toHaveTextContent("已复制")
    expect(card.queryByRole("alert")).not.toBeInTheDocument()
  })

  it("配置加载过程中禁止复制两个客户端的地址和配置", async () => {
    vi.mocked(getAppInfo).mockReturnValue(new Promise(() => {}))
    renderGateway("manual")
    await screen.findByRole("region", { name: "Codex 接口" })
    for (const client of ["Codex", "Claude"]) {
      const card = within(screen.getByRole("region", { name: `${client} 接口` }))
      expect(card.getByRole("button", { name: `复制 ${client} 地址` })).toBeDisabled()
      expect(card.getByRole("button", { name: `复制 ${client} 配置` })).toBeDisabled()
    }
  })

  it("复制进行中禁止重复提交当前动作", async () => {
    const user = userEvent.setup()
    vi.mocked(Clipboard.SetText).mockReturnValue(new Promise(() => {}))
    renderGateway("manual")
    await screen.findByText("http://127.0.0.1:43210/v1")
    const card = within(screen.getByRole("region", { name: "Codex 接口" }))
    await user.click(card.getByRole("button", { name: "复制 Codex 地址" }))
    const copying = card.getByRole("button", { name: "正在复制…" })
    expect(copying).toBeDisabled()
    await user.click(copying)
    expect(Clipboard.SetText).toHaveBeenCalledTimes(1)
  })

  it("已运行网关使用实际端口并标记协议已接入", async () => {
    vi.mocked(getGatewayInfo).mockResolvedValue({
      ...gatewayInfo,
      runtime: { ...gatewayInfo.runtime, state: "running", port: 54321 },
    })
    renderGateway("manual")
    await screen.findByText("运行中")
    const card = within(screen.getByRole("region", { name: "Claude 接口" }))
    expect(card.getByRole("textbox", { name: "Base URL" })).toHaveValue(
      "http://127.0.0.1:54321/claude",
    )
    expect(screen.getAllByText("协议已接入")).toHaveLength(2)
  })

  it("英文界面包含两个客户端入口和桌面说明", async () => {
    const user = userEvent.setup()
    renderGateway("manual")
    await screen.findByText("http://127.0.0.1:43210/v1")
    act(() => useUIStore.setState({ language: "en" }))
    const card = within(await screen.findByRole("region", { name: "Claude API" }))
    expect(screen.getByRole("region", { name: "Codex API" })).toBeInTheDocument()
    expect(screen.getAllByText("Protocol available")).toHaveLength(2)
    await user.click(card.getByRole("button", { name: "Desktop" }))
    expect(card.getByText(/Configure Third-Party Inference/)).toBeInTheDocument()
  })
})
