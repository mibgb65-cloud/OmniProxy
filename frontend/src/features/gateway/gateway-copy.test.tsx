import { Clipboard } from "@wailsio/runtime"
import { screen, waitFor } from "@testing-library/react"
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

describe("网关地址复制", () => {
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

  it("复制实际代理地址，不复制独立控制 API 地址", async () => {
    const user = userEvent.setup()
    renderGateway("manual")
    await screen.findByText("http://127.0.0.1:43210/v1")
    await user.click(screen.getByRole("button", { name: "复制配置地址" }))
    await waitFor(() => expect(Clipboard.SetText).toHaveBeenCalledWith("http://127.0.0.1:43210/v1"))
    expect(await screen.findByRole("status")).toHaveTextContent("已复制配置地址。")
  })

  it("复制失败显示错误，用户可以重试", async () => {
    const user = userEvent.setup()
    vi.mocked(Clipboard.SetText).mockRejectedValueOnce(new Error("clipboard unavailable"))
    renderGateway("manual")
    await screen.findByText("http://127.0.0.1:43210/v1")
    await user.click(screen.getByRole("button", { name: "复制配置地址" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("复制失败，请重试。")
    await user.click(screen.getByRole("button", { name: "复制配置地址" }))
    expect(await screen.findByRole("status")).toHaveTextContent("已复制配置地址。")
    expect(screen.queryByRole("alert")).not.toBeInTheDocument()
  })

  it("复制未完成时禁止重复提交", async () => {
    const user = userEvent.setup()
    vi.mocked(Clipboard.SetText).mockReturnValue(new Promise(() => {}))
    renderGateway("manual")
    await screen.findByText("http://127.0.0.1:43210/v1")
    await user.click(screen.getByRole("button", { name: "复制配置地址" }))
    expect(await screen.findByRole("button", { name: "正在复制…" })).toBeDisabled()
    expect(Clipboard.SetText).toHaveBeenCalledTimes(1)
  })
})
