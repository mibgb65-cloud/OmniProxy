import { act, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { getGatewayInfo, startGateway } from "../../services/gateway"
import { getAppInfo, listAccounts, listProviders } from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import { agentClientIds, type AgentClientId } from "./agent-clients"
import {
  accounts,
  agentErrorMock,
  applyAgentConfigMock,
  appInfo,
  gatewayInfo,
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
  Events: { On: vi.fn(() => () => {}) },
}))

const agent = {
  client: "codex",
  path: "test/config.toml",
  exists: true,
  configured: false,
  provider: "openai",
  canRestore: false,
  revision: "test-revision",
  errorCode: "",
}

const clientNames = {
  codex: "Codex CLI",
  claude: "Claude Code CLI",
  "claude-desktop": "Claude Code Desktop",
  opencode: "OpenCode CLI",
  "harness-web": "DeepSeek Harness Web",
  "harness-desktop": "DeepSeek Harness Desktop",
  kilocode: "Kilo Code CLI",
  "qwen-code": "Qwen Code CLI",
  "mimo-code": "MiMo Code CLI",
} as const

async function openClient(
  user: ReturnType<typeof userEvent.setup>,
  client: AgentClientId = "codex",
) {
  await user.click(await screen.findByRole("link", { name: `配置 ${clientNames[client]}` }))
}

beforeEach(() => {
  vi.clearAllMocks()
  resetGatewayRuntime()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getAppInfo).mockResolvedValue(appInfo)
  vi.mocked(listProviders).mockResolvedValue(providers)
  vi.mocked(listAccounts).mockResolvedValue(accounts)
  listAgentClientsMock.mockResolvedValue([agent])
  previewAgentPlanMock.mockResolvedValue({
    summary: "接入配置预览",
    mode: "local-catalog",
    defaultModel: "deepseek-flash",
    catalogRevision: "catalog-revision",
    models: (gatewayInfo.models ?? []).map((id) => ({
      id,
      name: id,
      targetModel: id,
      providerId: "p",
      providerKind: "deepseek",
      alias: false,
      metadata: {},
    })),
  })
  applyAgentConfigMock.mockResolvedValue([])
})

it.each(agentClientIds)(
  "网关停止时仍可预览并应用 %s 接入配置，不会自动启动网关",
  async (client) => {
    listAgentClientsMock.mockResolvedValue([{ ...agent, client }])
    const user = userEvent.setup()
    renderGateway("clients")
    await openClient(user, client)
    const configure = await screen.findByRole("button", { name: "一键配置" })
    expect(configure).toBeEnabled()
    expect(screen.getByText(/可以先保存接入配置/)).toBeInTheDocument()
    await user.click(configure)
    expect(screen.queryByRole("combobox")).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "配置预览" }))
    expect(await screen.findByText("将提供 2 个模型／路由")).toBeInTheDocument()
    const input = { client, revision: agent.revision }
    expect(previewAgentPlanMock).toHaveBeenCalledWith(input)
    await user.click(screen.getByRole("button", { name: "应用配置" }))
    await waitFor(() =>
      expect(applyAgentConfigMock).toHaveBeenCalledWith({
        ...input,
        catalogRevision: "catalog-revision",
      }),
    )
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("配置已保存"))
    await user.click(screen.getAllByRole("button", { name: "关闭" })[0]!)
    expect(screen.getByRole("button", { name: "启动网关" })).toBeEnabled()
    expect(startGateway).not.toHaveBeenCalled()
  },
)

it("停止状态下没有可用模型时保留配置限制，并提示模型准备步骤", async () => {
  const user = userEvent.setup()
  vi.mocked(getGatewayInfo).mockResolvedValue({ ...gatewayInfo, models: [] })
  renderGateway("clients")
  await openClient(user)
  expect(await screen.findByRole("button", { name: "一键配置" })).toBeDisabled()
  expect(screen.getByText(/在路由页同步账号模型或配置可用路由/)).toBeInTheDocument()
  expect(previewAgentPlanMock).not.toHaveBeenCalled()
  expect(applyAgentConfigMock).not.toHaveBeenCalled()
  expect(startGateway).not.toHaveBeenCalled()
})

it("停止状态下原配置异常时仍阻止覆盖", async () => {
  const user = userEvent.setup()
  listAgentClientsMock.mockResolvedValue([{ ...agent, errorCode: "invalid_config" }])
  renderGateway("clients")
  await openClient(user)
  expect(await screen.findByRole("button", { name: "一键配置" })).toBeDisabled()
  expect(applyAgentConfigMock).not.toHaveBeenCalled()
})

it("Qwen 历史设置需要官方迁移时提示先启动客户端，不允许应用", async () => {
  agentErrorMock.mockReturnValueOnce("client_upgrade_required")
  listAgentClientsMock.mockResolvedValue([{ ...agent, client: "qwen-code" }])
  previewAgentPlanMock.mockRejectedValueOnce(new Error("agent: client_upgrade_required"))
  const user = userEvent.setup()
  renderGateway("clients")
  await openClient(user, "qwen-code")
  await user.click(await screen.findByRole("button", { name: "一键配置" }))
  await user.click(screen.getByRole("button", { name: "配置预览" }))
  expect(
    await screen.findByText("请先启动当前版 Qwen Code，完成设置升级后重新读取配置。"),
  ).toBeInTheDocument()
  expect(screen.queryByRole("button", { name: "应用配置" })).not.toBeInTheDocument()
  expect(applyAgentConfigMock).not.toHaveBeenCalled()
})

it("英文提示也说明可先保存配置、调用前启动", async () => {
  const user = userEvent.setup()
  renderGateway("clients")
  await openClient(user)
  await screen.findByRole("button", { name: "一键配置" })
  act(() => useUIStore.setState({ language: "en" }))
  expect(await screen.findByRole("button", { name: "Configure" })).toBeEnabled()
  expect(screen.getByText(/You can save the connection configuration now/)).toHaveTextContent(
    "start the gateway before making model requests",
  )
})
