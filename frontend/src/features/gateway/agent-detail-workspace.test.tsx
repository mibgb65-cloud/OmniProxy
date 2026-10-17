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
  listAgentClientsMock,
  getConfiguredAgentModelsMock,
  previewAgentPlanMock,
  applyAgentConfigMock,
  restoreAgentConfigMock,
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

const item = {
  client: "codex",
  path: "test/config.toml",
  exists: true,
  configured: true,
  provider: "openai",
  canRestore: true,
  revision: "saved",
  errorCode: "",
}
beforeEach(() => {
  vi.clearAllMocks()
  resetGatewayRuntime()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getAppInfo).mockResolvedValue(appInfo)
  vi.mocked(listProviders).mockResolvedValue(providers)
  vi.mocked(listAccounts).mockResolvedValue(accounts)
  listAgentClientsMock.mockResolvedValue([item])
  getConfiguredAgentModelsMock.mockResolvedValue({
    models: [{ id: "saved", name: "Saved Model" }],
    defaultModel: "saved",
  })
  previewAgentPlanMock.mockResolvedValue({
    summary: "",
    mode: "local-catalog",
    defaultModel: "a",
    catalogRevision: "catalog",
    models: ["a", "b"].map((id) => ({
      id,
      name: `Model ${id}`,
      targetModel: id,
      providerId: "p",
      providerKind: "openai",
      alias: false,
      metadata: {},
    })),
  })
  applyAgentConfigMock.mockResolvedValue([])
})

it("右侧展示实际已接入模型，进入详情不发起配置预览或写入", async () => {
  renderGateway("clients", { entry: "/gateway?tab=clients&client=codex" })
  const heading = await screen.findByRole("heading", { name: "已接入模型" })
  const panel = heading.closest("section") ?? heading.closest("[data-slot='card']")!
  expect(await within(panel as HTMLElement).findByText("Saved Model")).toBeInTheDocument()
  expect(getConfiguredAgentModelsMock).toHaveBeenCalledWith("codex")
  expect(screen.queryByText("Model a")).not.toBeInTheDocument()
  expect(previewAgentPlanMock).not.toHaveBeenCalled()
  expect(applyAgentConfigMock).not.toHaveBeenCalled()
})

it("配置在右侧直接编辑，取消保留原模型并恢复按钮焦点，再进入仍默认全选", async () => {
  const user = userEvent.setup()
  renderGateway("clients", { entry: "/gateway?tab=clients&client=codex" })
  await user.click(await screen.findByRole("button", { name: "一键配置" }))
  expect(await screen.findByRole("checkbox", { name: "配置 Model a" })).toBeChecked()
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
  expect(screen.queryByRole("heading", { name: "已接入模型" })).not.toBeInTheDocument()
  expect(screen.getByRole("button", { name: "恢复原配置" })).toBeDisabled()
  await user.click(screen.getByRole("checkbox", { name: "配置 Model a" }))
  await user.click(screen.getAllByRole("button", { name: "取消配置" })[0]!)
  expect(await screen.findByText("Saved Model")).toBeInTheDocument()
  expect(screen.getByRole("button", { name: "一键配置" })).toHaveFocus()
  expect(applyAgentConfigMock).not.toHaveBeenCalled()
  await user.click(screen.getByRole("button", { name: "一键配置" }))
  expect(await screen.findByRole("checkbox", { name: "配置 Model a" })).toBeChecked()
})

it("保存部分模型后返回实际列表，恢复原配置后显示空状态", async () => {
  const user = userEvent.setup()
  applyAgentConfigMock.mockImplementation(async () => {
    listAgentClientsMock.mockResolvedValue([{ ...item, revision: "updated" }])
    getConfiguredAgentModelsMock.mockResolvedValue({
      models: [{ id: "b", name: "Model b" }],
      defaultModel: "b",
    })
    return []
  })
  restoreAgentConfigMock.mockImplementation(async () => {
    listAgentClientsMock.mockResolvedValue([
      { ...item, canRestore: false, configured: false, revision: "restored" },
    ])
    return []
  })
  renderGateway("clients", { entry: "/gateway?tab=clients&client=codex" })
  await user.click(await screen.findByRole("button", { name: "一键配置" }))
  await user.click(await screen.findByRole("checkbox", { name: "配置 Model a" }))
  await user.click(screen.getByRole("button", { name: "应用配置" }))
  expect(await screen.findByRole("heading", { name: "已接入模型" })).toBeInTheDocument()
  expect(await screen.findByText("Model b")).toBeInTheDocument()
  expect(screen.queryByText("Saved Model")).not.toBeInTheDocument()
  expect(applyAgentConfigMock).toHaveBeenCalledWith({
    client: "codex",
    revision: "saved",
    catalogRevision: "catalog",
    modelIds: ["b"],
    model: "b",
  })
  await user.click(screen.getByRole("button", { name: "恢复原配置" }))
  expect(await screen.findByText("尚未接入模型")).toBeInTheDocument()
  expect(screen.queryByText("Model b")).not.toBeInTheDocument()
})

it("切换客户端隔离模型列表", async () => {
  const user = userEvent.setup()
  listAgentClientsMock.mockResolvedValue([item, { ...item, client: "claude", revision: "claude" }])
  getConfiguredAgentModelsMock.mockImplementation(async (client) => ({
    models: [{ id: client, name: `${client} saved` }],
    defaultModel: client,
  }))
  renderGateway("clients", { entry: "/gateway?tab=clients&client=codex" })
  expect(await screen.findByText("codex saved")).toBeInTheDocument()
  await user.click(screen.getByRole("link", { name: "返回本地 Agent" }))
  await user.click(await screen.findByRole("link", { name: "配置 Claude Code CLI" }))
  expect(await screen.findByText("claude saved")).toBeInTheDocument()
  await waitFor(() => expect(screen.queryByText("codex saved")).not.toBeInTheDocument())
})
