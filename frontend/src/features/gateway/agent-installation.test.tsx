import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { getAppInfo, listAccounts, listProviders } from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import {
  accounts,
  appInfo,
  providers,
  renderGateway,
  resetGatewayRuntime,
  getAgentInstallationsMock,
  refreshAgentInstallationsMock,
  installAgentMock,
  openAgentInstallationOfficialMock,
  openClaudeCodeMock,
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

const missing = {
  items: ["codex-cli", "codex-desktop", "claude-code", "claude-desktop"].map((id) => ({
    id,
    detection: { state: "not_installed", version: "", path: "", errorCode: "" },
    installState: "idle",
    errorCode: "",
    officialURL: "https://example.invalid",
  })),
  busy: false,
  installerAvailable: true,
}

beforeEach(() => {
  vi.clearAllMocks()
  resetGatewayRuntime()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getAppInfo).mockResolvedValue(appInfo)
  vi.mocked(listProviders).mockResolvedValue(providers)
  vi.mocked(listAccounts).mockResolvedValue(accounts)
  listAgentClientsMock.mockResolvedValue([])
  getAgentInstallationsMock.mockResolvedValue(missing)
})

it("打开详情只检测，Codex CLI 与桌面应用分别提供安装入口", async () => {
  renderGateway("clients", { entry: "/gateway?tab=clients&client=codex" })
  expect(await screen.findByRole("button", { name: "安装 Codex CLI" })).toBeEnabled()
  expect(screen.getByRole("button", { name: "安装 Codex Desktop" })).toBeEnabled()
  expect(installAgentMock).not.toHaveBeenCalled()
  expect(openAgentInstallationOfficialMock).not.toHaveBeenCalled()
})

it("安装只提交固定 ID，进行中禁止重复安装，轮询真实结果后展示版本", async () => {
  const running = {
    ...missing,
    busy: true,
    items: missing.items.map((item) =>
      item.id === "claude-code" ? { ...item, installState: "installing" } : item,
    ),
  }
  const completed = {
    ...missing,
    items: missing.items.map((item) =>
      item.id === "claude-code"
        ? {
            ...item,
            installState: "completed",
            detection: { ...item.detection, state: "installed", version: "2.1.99" },
          }
        : item,
    ),
  }
  getAgentInstallationsMock.mockResolvedValueOnce(missing).mockResolvedValue(completed)
  installAgentMock.mockResolvedValue(running)
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=claude" })
  await user.click(await screen.findByRole("button", { name: "安装 Claude Code CLI" }))
  expect(installAgentMock).toHaveBeenCalledWith("claude-code")
  expect(installAgentMock).toHaveBeenCalledTimes(1)
  expect(screen.getByRole("button", { name: "安装 Claude Code CLI" })).toBeDisabled()
  expect(await screen.findByText("已安装 · 2.1.99", {}, { timeout: 5000 })).toBeInTheDocument()
  expect(screen.queryByRole("button", { name: "安装 Claude Code CLI" })).not.toBeInTheDocument()
})

it("没有 WinGet 时阻止安装，并支持固定官方说明入口", async () => {
  getAgentInstallationsMock.mockResolvedValue({ ...missing, installerAvailable: false })
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=claude-desktop" })
  expect(await screen.findByRole("button", { name: "安装 Claude Code Desktop" })).toBeDisabled()
  expect(screen.getByText(/未找到 WinGet/)).toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "Claude Code Desktop 官方安装说明" }))
  await waitFor(() =>
    expect(openAgentInstallationOfficialMock).toHaveBeenCalledWith("claude-desktop"),
  )
  expect(installAgentMock).not.toHaveBeenCalled()
})

it("Claude Code Desktop 说明共用安装包，已安装时提供 Code 编程入口", async () => {
  getAgentInstallationsMock.mockResolvedValue({
    ...missing,
    items: missing.items.map((item) => ({
      ...item,
      detection: { ...item.detection, state: "installed" },
    })),
  })
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=claude-desktop" })
  expect(await screen.findByText(/与 Chat 共用安装包/)).toBeInTheDocument()
  expect(screen.queryByRole("button", { name: "安装 Claude Code Desktop" })).not.toBeInTheDocument()
  await user.click(await screen.findByRole("button", { name: "打开 Claude Code" }))
  expect(openClaudeCodeMock).toHaveBeenCalledTimes(1)
  expect(installAgentMock).not.toHaveBeenCalled()
})

it("检测异常不会被当作未安装，重新检测成功后允许安装", async () => {
  getAgentInstallationsMock.mockResolvedValue({
    ...missing,
    items: missing.items.map((item) => ({
      ...item,
      detection: { ...item.detection, state: "unknown", errorCode: "detection_failed" },
    })),
  })
  refreshAgentInstallationsMock.mockResolvedValue(missing)
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=claude" })
  expect(await screen.findByRole("button", { name: "安装 Claude Code CLI" })).toBeDisabled()
  await user.click(screen.getByRole("button", { name: "重新检测" }))
  await waitFor(() =>
    expect(screen.getByRole("button", { name: "安装 Claude Code CLI" })).toBeEnabled(),
  )
  expect(installAgentMock).not.toHaveBeenCalled()
})

it("安装错误不展示原始输出，并保留重试入口", async () => {
  installAgentMock.mockRejectedValue(new Error("private installer token"))
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=claude" })
  await user.click(await screen.findByRole("button", { name: "安装 Claude Code CLI" }))
  const card = screen.getByRole("heading", { name: "软件安装" }).closest('[data-slot="card"]')!
  expect(await within(card as HTMLElement).findByRole("alert")).toHaveTextContent("操作失败")
  expect(screen.queryByText("private installer token")).not.toBeInTheDocument()
  await waitFor(() =>
    expect(screen.getByRole("button", { name: "安装 Claude Code CLI" })).toBeEnabled(),
  )
})
