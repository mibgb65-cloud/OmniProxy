import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { getAppInfo, listAccounts, listProviders } from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import { agentClientIds } from "./agent-clients"
import {
  accounts,
  appInfo,
  applyAgentConfigMock,
  getAgentInstallationsMock,
  listAgentClientsMock,
  previewAgentPlanMock,
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

beforeEach(() => {
  vi.clearAllMocks()
  resetGatewayRuntime()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getAppInfo).mockResolvedValue(appInfo)
  vi.mocked(listProviders).mockResolvedValue(providers)
  vi.mocked(listAccounts).mockResolvedValue(accounts)
  listAgentClientsMock.mockResolvedValue(
    agentClientIds.map((client) => ({
      client,
      path: "test/client.json",
      exists: true,
      configured: true,
      provider: "",
      canRestore: true,
      revision: "revision",
      errorCode: "",
    })),
  )
})

it("所有产品使用标题分组，内部按软件列出配置入口", async () => {
  renderGateway("clients")
  for (const [title, labels] of [
    ["Codex", ["Codex CLI", "Codex Desktop"]],
    ["Claude Code", ["Claude Code CLI", "Claude Code Desktop"]],
    ["OpenCode", ["OpenCode CLI", "OpenCode Desktop"]],
    ["DeepSeek Harness", ["DeepSeek Harness Desktop", "DeepSeek Harness Web"]],
    ["Kilo Code", ["Kilo Code CLI", "Kilo Desktop"]],
    ["Qwen Code", ["Qwen Code CLI", "Qwen Code Desktop"]],
    ["MiMo Code", ["MiMo Code CLI", "MiMo Desktop 国内版", "MiMo Desktop 国际版"]],
    ["Kimi Code", ["Kimi Code CLI"]],
    ["Pi", ["Pi CLI"]],
    ["Factory Droid", ["Factory Droid CLI"]],
    ["Goose", ["Goose CLI", "Goose Desktop"]],
  ] as const) {
    const group = await screen.findByRole("region", { name: title })
    expect(within(group).getByRole("heading", { name: title, level: 3 })).toBeInTheDocument()
    expect(within(group).getAllByRole("link")).toHaveLength(labels.length)
    for (const label of labels) {
      expect(within(group).getByRole("link", { name: `配置 ${label}` })).toBeInTheDocument()
    }
  }
  expect(previewAgentPlanMock).not.toHaveBeenCalled()
  expect(applyAgentConfigMock).not.toHaveBeenCalled()
})

it.each([
  ["codex", "Codex Desktop", "Codex CLI / Desktop"],
  ["opencode", "OpenCode Desktop", "OpenCode CLI / Desktop"],
  ["kilocode", "Kilo Desktop", "Kilo Code CLI / Desktop"],
  ["qwen-code", "Qwen Code Desktop", "Qwen Code CLI / Desktop"],
  ["mimo-code", "MiMo Desktop 国内版", "MiMo Code CLI / Desktop"],
  ["mimo-code", "MiMo Desktop 国际版", "MiMo Code CLI / Desktop"],
  ["goose", "Goose Desktop", "Goose CLI / Desktop"],
])("%s 的 %s 进入共享详情，并返回原软件入口", async (client, label, heading) => {
  const user = userEvent.setup()
  const { router } = renderGateway("clients")
  const entry = await screen.findByRole("link", { name: `配置 ${label}` })
  entry.focus()
  await user.keyboard("{Enter}")
  expect(await screen.findByRole("heading", { name: heading, level: 2 })).toHaveFocus()
  expect(router.state.location.search).toEqual({ tab: "clients", client })
  await user.click(screen.getByRole("link", { name: "返回本地 Agent" }))
  await waitFor(() => expect(screen.getByRole("link", { name: `配置 ${label}` })).toHaveFocus())
  expect(applyAgentConfigMock).not.toHaveBeenCalled()
})

it("共享配置的 Key 状态一致，每个软件单独展示真实安装状态", async () => {
  getAgentInstallationsMock.mockResolvedValue({
    items: ["mimo-code-cli", "mimo-desktop-cn", "mimo-desktop-global"].map((id) => ({
      id,
      detection: { state: id === "mimo-desktop-global" ? "not_installed" : "installed" },
      installState: "idle",
    })),
    busy: false,
    installerAvailable: true,
  })
  renderGateway("clients")
  for (const [label, state] of [
    ["MiMo Code CLI", "已安装"],
    ["MiMo Desktop 国内版", "已安装"],
    ["MiMo Desktop 国际版", "未安装"],
  ]) {
    const entry = await screen.findByRole("link", { name: `配置 ${label}` })
    expect(await within(entry).findByText(state)).toBeInTheDocument()
    expect(within(entry).getByText("接入 Key 已启用")).toBeInTheDocument()
    expect(
      within(entry).queryByText(state === "已安装" ? "未安装" : "已安装"),
    ).not.toBeInTheDocument()
  }
})

it("返回入口的焦点滚动发生在父页面滚动归零之后", async () => {
  const user = userEvent.setup()
  renderGateway("clients", { scrollViewport: true })
  await user.click(await screen.findByRole("link", { name: "配置 OpenCode Desktop" }))
  await screen.findByRole("heading", { name: "OpenCode CLI / Desktop", level: 2 })
  const viewport = screen.getByRole("main")
  viewport.scrollTop = 40
  // jsdom 没有布局；模拟浏览器将屏幕外的焦点入口滚入视口。
  viewport.addEventListener("focusin", (event) => {
    if ((event.target as HTMLElement).getAttribute("aria-label") === "配置 OpenCode Desktop") {
      viewport.scrollTop = 360
    }
  })
  await user.click(screen.getByRole("link", { name: "返回本地 Agent" }))
  await waitFor(() =>
    expect(screen.getByRole("link", { name: "配置 OpenCode Desktop" })).toHaveFocus(),
  )
  expect(viewport.scrollTop).toBe(360)
})
