import { act, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { getAppInfo, listAccounts, listProviders } from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import {
  accounts,
  applyAgentConfigMock,
  appInfo,
  copyAgentKeyMock,
  listAgentClientsMock,
  previewAgentPlanMock,
  providers,
  renderGateway,
  resetGatewayRuntime,
  restoreAgentConfigMock,
  setAgentEnabledMock,
  getAgentInstallationsMock,
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

const codex = {
  client: "codex",
  path: "test/config.toml",
  exists: true,
  configured: true,
  provider: "openai",
  canRestore: true,
  revision: "test-revision",
  errorCode: "",
}

beforeEach(() => {
  vi.clearAllMocks()
  resetGatewayRuntime()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getAppInfo).mockResolvedValue(appInfo)
  vi.mocked(listProviders).mockResolvedValue(providers)
  vi.mocked(listAccounts).mockResolvedValue(accounts)
  listAgentClientsMock.mockResolvedValue([
    codex,
    { ...codex, client: "claude", configured: false, canRestore: false },
    { ...codex, client: "claude-desktop", configured: false },
  ])
})

it("入口按 Codex 与 Claude 归组，点击仅导航，前进后退恢复详情与卡片焦点", async () => {
  const user = userEvent.setup()
  const { router } = renderGateway("clients")
  const tile = await screen.findByRole("link", { name: "配置 Codex CLI" })
  const claude = screen.getByRole("region", { name: "Claude Code" })
  expect(within(claude).getAllByRole("link")).toHaveLength(2)
  expect(within(claude).getByRole("link", { name: "配置 Claude Code CLI" })).toBeInTheDocument()
  expect(within(claude).getByRole("link", { name: "配置 Claude Code Desktop" })).toBeInTheDocument()
  expect(screen.queryByText(codex.path)).not.toBeInTheDocument()
  expect(screen.queryByRole("button", { name: "一键配置" })).not.toBeInTheDocument()
  tile.focus()
  await user.keyboard("{Enter}")
  const heading = await screen.findByRole("heading", { name: "Codex CLI / Desktop" })
  expect(heading).toHaveFocus()
  expect(screen.getByText(codex.path)).toBeInTheDocument()
  expect(router.state.location.search).toEqual({ tab: "clients", client: "codex" })
  expect(previewAgentPlanMock).not.toHaveBeenCalled()
  expect(applyAgentConfigMock).not.toHaveBeenCalled()
  await user.click(screen.getByRole("link", { name: "返回本地 Agent" }))
  await waitFor(() => expect(screen.getByRole("link", { name: "配置 Codex CLI" })).toHaveFocus())
  await act(async () => router.history.back())
  expect(await screen.findByRole("heading", { name: "Codex CLI / Desktop" })).toBeInTheDocument()
  await act(async () => router.history.forward())
  expect(await screen.findByRole("link", { name: "配置 Codex CLI" })).toBeInTheDocument()
})

it.each([
  ["claude", "Claude Code CLI"],
  ["claude-desktop", "Claude Code Desktop"],
])("Claude 分组中的 %s 保留独立键盘入口和返回焦点", async (client, title) => {
  const user = userEvent.setup()
  const { router } = renderGateway("clients")
  const entry = await screen.findByRole("link", { name: `配置 ${title}` })
  entry.focus()
  await user.keyboard("{Enter}")
  expect(await screen.findByRole("heading", { name: title, level: 2 })).toHaveFocus()
  expect(router.state.location.search).toEqual({ tab: "clients", client })
  expect(applyAgentConfigMock).not.toHaveBeenCalled()
  await user.click(screen.getByRole("link", { name: "返回本地 Agent" }))
  await waitFor(() => expect(screen.getByRole("link", { name: `配置 ${title}` })).toHaveFocus())
})

it("Claude 分组分别展示 CLI 与 Desktop 的安装和接入状态", async () => {
  getAgentInstallationsMock.mockResolvedValue({
    items: ["claude-code", "claude-desktop"].map((id) => ({
      id,
      detection: { state: id === "claude-code" ? "installed" : "not_installed" },
      installState: "idle",
    })),
    busy: false,
    installerAvailable: true,
  })
  renderGateway("clients")
  const cli = await screen.findByRole("link", { name: "配置 Claude Code CLI" })
  const desktop = screen.getByRole("link", { name: "配置 Claude Code Desktop" })
  expect(await within(cli).findByText("已安装")).toBeInTheDocument()
  expect(within(cli).getByText("尚未配置")).toBeInTheDocument()
  expect(within(desktop).getByText("未安装")).toBeInTheDocument()
  expect(within(desktop).getByText("接入 Key 已停用")).toBeInTheDocument()
})

it("深链接直接打开客户端，切换选项卡后保留选择", async () => {
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=claude-desktop" })
  expect(
    await screen.findByRole("heading", { name: "Claude Code Desktop", level: 2 }),
  ).toBeInTheDocument()
  expect(await screen.findByRole("button", { name: "一键配置" })).toBeEnabled()
  await user.click(screen.getByRole("tab", { name: "手动接入" }))
  expect(
    screen.queryByRole("heading", { name: "Claude Code Desktop", level: 2 }),
  ).not.toBeInTheDocument()
  await user.click(screen.getByRole("tab", { name: "客户端接入" }))
  expect(screen.getByRole("heading", { name: "Claude Code Desktop", level: 2 })).toBeInTheDocument()
})

it.each(["unknown", "123", "%5B%22codex%22%5D", "../../codex", "x".repeat(300)])(
  "非法客户端参数 %s 回到目录，不执行配置",
  async (client) => {
    renderGateway("clients", { entry: `/gateway?tab=clients&client=${client}` })
    expect(await screen.findByRole("heading", { name: "本地 Agent 接入" })).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "一键配置" })).not.toBeInTheDocument()
    expect(applyAgentConfigMock).not.toHaveBeenCalled()
  },
)

it("读取失败提供刷新提示，不使用旧数据开放配置操作", async () => {
  listAgentClientsMock.mockRejectedValueOnce(new Error("private client path"))
  renderGateway("clients", { entry: "/gateway?tab=clients&client=codex" })
  expect(await screen.findByText("无法读取客户端配置，请刷新后重试。")).toBeInTheDocument()
  expect(screen.queryByText("private client path")).not.toBeInTheDocument()
  expect(screen.queryByRole("button", { name: "一键配置" })).not.toBeInTheDocument()
})

it("复制、启停和恢复只操作详情页的客户端", async () => {
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=codex" })
  await user.click(await screen.findByRole("button", { name: "复制接入 Key" }))
  expect(await screen.findByRole("status")).toHaveTextContent("接入 Key 已复制")
  expect(copyAgentKeyMock).toHaveBeenCalledWith("codex")
  await user.click(screen.getByRole("button", { name: "停用接入 Key" }))
  await waitFor(() => expect(setAgentEnabledMock).toHaveBeenCalledWith("codex", false))
  await waitFor(() => expect(screen.getByRole("button", { name: "恢复原配置" })).toBeEnabled())
  await user.click(screen.getByRole("button", { name: "恢复原配置" }))
  expect(await screen.findByRole("status")).toHaveTextContent("已恢复原配置")
  expect(restoreAgentConfigMock).toHaveBeenCalledWith("codex")
})
