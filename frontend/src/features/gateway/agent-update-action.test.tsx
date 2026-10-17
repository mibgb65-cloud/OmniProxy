import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { getAppInfo, listAccounts, listProviders } from "../../services/omniproxy"
import type { AgentInstallation, AgentInstallSnapshot } from "../../services/agent-installation"
import { useUIStore } from "../../stores/ui-store"
import {
  accounts,
  appInfo,
  providers,
  renderGateway,
  resetGatewayRuntime,
  checkAgentUpdateMock,
  updateAgentMock,
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

const installedItem: AgentInstallation = {
  id: "claude-desktop",
  detection: { state: "installed", version: "2.9.0.0", path: "", errorCode: "" },
  installState: "idle",
  errorCode: "",
  officialURL: "https://example.invalid",
  canInstall: true,
}
const installed: AgentInstallSnapshot = {
  items: [installedItem],
  busy: false,
  installerAvailable: true,
}
function snapshot(installState: string): AgentInstallSnapshot {
  return {
    ...installed,
    busy: installState === "installing",
    items: [
      {
        ...installedItem,
        operation: "update",
        updateVersion: "2.10.0",
        installState,
        detection: {
          ...installedItem.detection,
          version: installState === "completed" ? "2.10.0.0" : "2.9.0.0",
        },
      },
    ],
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  resetGatewayRuntime()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getAppInfo).mockResolvedValue(appInfo)
  vi.mocked(listProviders).mockResolvedValue(providers)
  vi.mocked(listAccounts).mockResolvedValue(accounts)
  listAgentClientsMock.mockResolvedValue([])
  getAgentInstallationsMock.mockResolvedValue(installed)
  checkAgentUpdateMock.mockResolvedValue({
    state: "available",
    currentVersion: "2.9.0.0",
    latestVersion: "2.10.0",
  })
})

it("发现新版后原按钮变为更新，只有再次点击才更新；更新中不能重复提交，成功显示真实版本", async () => {
  let resolve!: (value: AgentInstallSnapshot) => void
  updateAgentMock.mockReturnValue(
    new Promise<AgentInstallSnapshot>((done) => {
      resolve = done
    }),
  )
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=claude-desktop" })
  await user.click(await screen.findByRole("button", { name: "检查更新" }))
  const button = await screen.findByRole("button", { name: "更新" })
  expect(updateAgentMock).not.toHaveBeenCalled()
  await user.click(button)
  expect(updateAgentMock).toHaveBeenCalledExactlyOnceWith("claude-desktop")
  expect(screen.getByRole("button", { name: "更新中…" })).toBeDisabled()
  await user.click(button)
  expect(updateAgentMock).toHaveBeenCalledTimes(1)
  resolve(snapshot("completed"))
  expect(await screen.findByText("已更新至 2.10.0.0")).toBeInTheDocument()
  expect(screen.getByRole("button", { name: "检查更新" })).toBeEnabled()
  expect(installAgentMock).not.toHaveBeenCalled()
})

it("后台更新展示进度，完成轮询后自动恢复检查按钮", async () => {
  updateAgentMock.mockResolvedValue(snapshot("installing"))
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=claude-desktop" })
  await user.click(await screen.findByRole("button", { name: "检查更新" }))
  await user.click(await screen.findByRole("button", { name: "更新" }))
  expect(
    await screen.findByRole("progressbar", { name: "Claude Code Desktop 安装进度" }),
  ).toBeInTheDocument()
  expect(screen.getByRole("button", { name: "更新中…" })).toBeDisabled()
  getAgentInstallationsMock.mockResolvedValue(snapshot("completed"))
  expect(await screen.findByText("已更新至 2.10.0.0", {}, { timeout: 3000 })).toBeInTheDocument()
  expect(screen.getByRole("button", { name: "检查更新" })).toBeEnabled()
})

it("启动失败保留更新入口供重试，不暴露原始错误", async () => {
  updateAgentMock.mockRejectedValue(new Error("private installer output"))
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=claude-desktop" })
  await user.click(await screen.findByRole("button", { name: "检查更新" }))
  await user.click(await screen.findByRole("button", { name: "更新" }))
  await waitFor(() => expect(screen.getByRole("button", { name: "更新" })).toBeEnabled())
  expect(screen.getByText("操作失败，请重新检测后重试。")).toHaveAttribute("role", "alert")
  expect(screen.queryByText(/private installer output/)).not.toBeInTheDocument()
})

it("没有系统安装器时仍能检查，发现版本后禁用更新操作", async () => {
  getAgentInstallationsMock.mockResolvedValue({ ...installed, installerAvailable: false })
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=claude-desktop" })
  await user.click(await screen.findByRole("button", { name: "检查更新" }))
  expect(await screen.findByRole("button", { name: "更新" })).toBeDisabled()
  expect(updateAgentMock).not.toHaveBeenCalled()
})
