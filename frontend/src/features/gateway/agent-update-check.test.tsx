import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { getAppInfo, listAccounts, listProviders } from "../../services/omniproxy"
import type { AgentUpdateCheck } from "../../services/agent-installation"
import { useUIStore } from "../../stores/ui-store"
import {
  accounts,
  appInfo,
  providers,
  renderGateway,
  resetGatewayRuntime,
  checkAgentUpdateMock,
  getAgentInstallationsMock,
  installAgentMock,
  listAgentClientsMock,
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

beforeEach(() => {
  vi.clearAllMocks()
  resetGatewayRuntime()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getAppInfo).mockResolvedValue(appInfo)
  vi.mocked(listProviders).mockResolvedValue(providers)
  vi.mocked(listAccounts).mockResolvedValue(accounts)
  listAgentClientsMock.mockResolvedValue([])
  checkAgentUpdateMock.mockResolvedValue({
    state: "current",
    currentVersion: "1.2.3",
    latestVersion: "1.2.3",
  })
})

it("按需检查并显示真实版本；加载只影响检查按钮，重复点击不重复查询", async () => {
  let resolve!: (value: AgentUpdateCheck) => void
  checkAgentUpdateMock.mockReturnValue(
    new Promise<AgentUpdateCheck>((done) => {
      resolve = done
    }),
  )
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=claude-desktop" })
  const button = await screen.findByRole("button", { name: "检查更新" })
  expect(checkAgentUpdateMock).not.toHaveBeenCalled()
  await user.click(button)
  expect(checkAgentUpdateMock).toHaveBeenCalledExactlyOnceWith("claude-desktop")
  expect(screen.getByRole("button", { name: "检查中…" })).toBeDisabled()
  expect(button).toHaveAttribute("aria-busy", "true")
  expect(screen.getByRole("button", { name: "打开 Claude Code" })).toBeEnabled()
  expect(screen.getByRole("button", { name: "Claude Code Desktop 官方安装说明" })).toBeEnabled()
  await user.click(button)
  expect(checkAgentUpdateMock).toHaveBeenCalledTimes(1)
  resolve({ state: "available", currentVersion: "2.9.0.0", latestVersion: "2.10.0" })
  expect(await screen.findByText("发现新版本 2.10.0")).toHaveAttribute("role", "status")
  expect(screen.getByText("已安装 · 2.9.0.0")).toBeInTheDocument()
  expect(screen.getByRole("button", { name: "检查更新" })).toBeEnabled()
  expect(installAgentMock).not.toHaveBeenCalled()
})

it.each([
  ["current", "暂无新版本"],
  ["unknown", "暂时无法确认最新版本，请查看官方说明。"],
])("%s 结果与无法确认的结果分别显示", async (state, message) => {
  checkAgentUpdateMock.mockResolvedValue({ state, currentVersion: "1.2.3", latestVersion: "1.2.3" })
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=claude-desktop" })
  await user.click(await screen.findByRole("button", { name: "检查更新" }))
  expect(await screen.findByText(message)).toBeInTheDocument()
  expect(installAgentMock).not.toHaveBeenCalled()
})

it("失败不暴露底层响应，可重试，重新检测安装状态后清除旧更新结果", async () => {
  checkAgentUpdateMock.mockRejectedValueOnce(new Error("private transport data"))
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=claude-desktop" })
  await user.click(await screen.findByRole("button", { name: "检查更新" }))
  expect(await screen.findByText("检查更新失败，请稍后重试。")).toBeInTheDocument()
  expect(screen.queryByText(/private transport data/)).not.toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "检查更新" }))
  expect(await screen.findByText("暂无新版本")).toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "重新检测" }))
  await waitFor(() => expect(screen.queryByText("暂无新版本")).not.toBeInTheDocument())
})

it("未安装不提供检查入口，也不会查询更新源", async () => {
  getAgentInstallationsMock.mockResolvedValue({
    items: [
      {
        id: "claude-desktop",
        detection: { state: "not_installed", version: "" },
        installState: "idle",
      },
    ],
    busy: false,
    installerAvailable: true,
  })
  renderGateway("clients", { entry: "/gateway?tab=clients&client=claude-desktop" })
  expect(await screen.findByText("未安装")).toBeInTheDocument()
  expect(screen.queryByRole("button", { name: "检查更新" })).not.toBeInTheDocument()
  expect(checkAgentUpdateMock).not.toHaveBeenCalled()
})

it("英文检查具有明确名称及独立结果", async () => {
  useUIStore.setState({ language: "en" })
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=claude-desktop" })
  await user.click(await screen.findByRole("button", { name: "Check for updates" }))
  expect(await screen.findByText("No newer version available")).toBeInTheDocument()
})
