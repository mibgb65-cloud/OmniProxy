import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { getGatewayInfo } from "../../services/gateway"
import { getAppInfo, listAccounts, listProviders } from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import {
  accounts,
  agentErrorMock,
  appInfo,
  applyAgentConfigMock,
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
const plan = {
  summary: "写入完整目录",
  mode: "local-catalog",
  defaultModel: "deepseek-flash",
  models: [{ id: "deepseek-flash", name: "deepseek-flash" }],
  catalogRevision: "catalog-revision",
}

beforeEach(() => {
  vi.clearAllMocks()
  resetGatewayRuntime()
  agentErrorMock.mockReturnValue("failed")
  previewAgentPlanMock.mockResolvedValue(plan)
  applyAgentConfigMock.mockResolvedValue([])
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

it("分组一键配置为分组内每个客户端写入完整目录，不要求手动选择模型", async () => {
  const user = userEvent.setup()
  renderGateway("clients")
  const button = await screen.findByRole("button", { name: "一键配置 Claude Code 全部模型" })
  await waitFor(() => expect(button).toBeEnabled())
  await user.click(button)
  await waitFor(() =>
    expect(applyAgentConfigMock).toHaveBeenCalledWith({
      client: "claude-desktop",
      revision: "test-revision",
      catalogRevision: "catalog-revision",
    }),
  )
  expect(previewAgentPlanMock).toHaveBeenCalledWith({ client: "claude", revision: "test-revision" })
  expect(applyAgentConfigMock).toHaveBeenCalledWith({
    client: "claude",
    revision: "test-revision",
    catalogRevision: "catalog-revision",
  })
  expect(applyAgentConfigMock).toHaveBeenCalledTimes(2)
  expect(await screen.findByText("已完成 2/2 个客户端的一键配置。")).toBeInTheDocument()
})

it("共享配置文件的客户端只写入一次", async () => {
  const user = userEvent.setup()
  renderGateway("clients")
  const button = await screen.findByRole("button", { name: "一键配置 Codex 全部模型" })
  await waitFor(() => expect(button).toBeEnabled())
  await user.click(button)
  await waitFor(() =>
    expect(screen.getByText("已完成 1/1 个客户端的一键配置。")).toBeInTheDocument(),
  )
  expect(applyAgentConfigMock).toHaveBeenCalledTimes(1)
  expect(applyAgentConfigMock).toHaveBeenCalledWith({
    client: "codex",
    revision: "test-revision",
    catalogRevision: "catalog-revision",
  })
})

it("单个客户端失败时保留其他结果并列出原因", async () => {
  const user = userEvent.setup()
  agentErrorMock.mockReturnValue("desktop_profile_active")
  applyAgentConfigMock.mockImplementation(async (input: { client: string }) => {
    if (input.client === "claude-desktop") throw new Error("agent: desktop_profile_active")
    return []
  })
  renderGateway("clients")
  const button = await screen.findByRole("button", { name: "一键配置 Claude Code 全部模型" })
  await waitFor(() => expect(button).toBeEnabled())
  await user.click(button)
  expect(await screen.findByText("已完成 1/2 个客户端的一键配置。")).toBeInTheDocument()
  expect(
    screen.getByText(
      "Claude Code Desktop：请先在 Claude Desktop 切回原配置，再移除当前正在使用的网关配置。",
    ),
  ).toBeInTheDocument()
})

it("没有可用模型路由时禁用一键配置并说明原因", async () => {
  vi.mocked(getGatewayInfo).mockResolvedValue({ ...gatewayInfo, models: [] })
  renderGateway("clients")
  const button = await screen.findByRole("button", { name: "一键配置 Claude Code 全部模型" })
  expect(button).toBeDisabled()
  expect(screen.getAllByText("暂无可用模型路由").length).toBeGreaterThan(0)
  expect(applyAgentConfigMock).not.toHaveBeenCalled()
})
