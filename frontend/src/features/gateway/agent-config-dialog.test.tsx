import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import {
  agentError,
  applyAgentConfig,
  previewAgentPlan,
  rereadAgentConfig,
} from "../../services/agents"
import { AgentConfigDialog } from "./agent-config-dialog"
import { useUIStore } from "../../stores/ui-store"

vi.mock("../../services/agents", () => ({
  previewAgentPlan: vi.fn(),
  applyAgentConfig: vi.fn(),
  rereadAgentConfig: vi.fn(),
  agentError: vi.fn(() => "config_changed"),
  agentTitle: () => "agents.codex",
}))
const item = {
  client: "codex",
  path: "test/config.toml",
  exists: true,
  configured: false,
  provider: "openai",
  canRestore: false,
  revision: "test-revision",
  errorCode: "",
}
function mount() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <AgentConfigDialog item={item} onClose={vi.fn()} />
      </QueryClientProvider>
    </I18nProvider>,
  )
}
beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(agentError).mockReturnValue("config_changed")
  vi.mocked(previewAgentPlan).mockResolvedValue({
    summary: "保留 model_provider=openai",
    mode: "local-catalog",
    defaultModel: "gpt-test",
    catalogRevision: "catalog-revision",
    models: [
      {
        id: "gpt-test",
        name: "GPT Test",
        targetModel: "gpt-test",
        providerId: "p",
        providerKind: "openai",
        alias: false,
        metadata: {},
      },
    ],
  })
  vi.mocked(applyAgentConfig).mockResolvedValue([])
})
it("无需选择模型即可预览完整目录、应用并使用同一文件和目录版本", async () => {
  const user = userEvent.setup()
  mount()
  expect(screen.getByRole("dialog")).toBeInTheDocument()
  expect(screen.getByRole("button", { name: "配置预览" })).toBeEnabled()
  expect(screen.queryByRole("combobox")).not.toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "配置预览" }))
  expect(await screen.findByText("GPT Test")).toBeInTheDocument()
  expect(screen.getByText("将提供 1 个模型／路由")).toBeInTheDocument()
  expect(applyAgentConfig).not.toHaveBeenCalled()
  await user.click(screen.getByRole("button", { name: "应用配置" }))
  await waitFor(() =>
    expect(applyAgentConfig).toHaveBeenCalledWith({
      client: "codex",
      revision: "test-revision",
      catalogRevision: "catalog-revision",
    }),
  )
  await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("配置已保存"))
  const notice = screen
    .getByRole("button", { name: "关闭保存成功提示" })
    .closest("[data-slot='agent-config-success-toast']")!
  expect(notice.closest(".scroll-region")).toBeNull()
  expect(screen.getByRole("dialog")).toContainElement(notice as HTMLElement)
  await user.click(screen.getByRole("button", { name: "关闭保存成功提示" }))
  expect(screen.queryByText("配置已保存，原配置已加密备份。")).not.toBeInTheDocument()
  expect(screen.queryByRole("button", { name: "应用配置" })).not.toBeInTheDocument()
  expect(screen.queryByRole("button", { name: "配置预览" })).not.toBeInTheDocument()
})
it("配置被外部修改时保持弹窗并展示安全错误", async () => {
  vi.mocked(applyAgentConfig).mockRejectedValue(new Error("private raw config"))
  const user = userEvent.setup()
  mount()
  await user.click(screen.getByRole("button", { name: "配置预览" }))
  await user.click(await screen.findByRole("button", { name: "应用配置" }))
  expect(await screen.findByRole("alert")).toHaveTextContent("重新读取配置")
  expect(screen.queryByRole("button", { name: "关闭保存成功提示" })).not.toBeInTheDocument()
  expect(screen.queryByText("private raw config")).not.toBeInTheDocument()
})
it("目录更新后回到预览步骤，允许重新读取后再应用", async () => {
  vi.mocked(agentError).mockReturnValue("catalog_changed")
  vi.mocked(applyAgentConfig).mockRejectedValueOnce(new Error("agent: catalog_changed"))
  const user = userEvent.setup()
  mount()
  await user.click(screen.getByRole("button", { name: "配置预览" }))
  await user.click(await screen.findByRole("button", { name: "应用配置" }))
  expect(await screen.findByRole("alert")).toHaveTextContent("请重新预览后应用")
  await user.click(screen.getByRole("button", { name: "配置预览" }))
  await waitFor(() => expect(previewAgentPlan).toHaveBeenCalledTimes(2))
})

it("外部修改后重新读取最新版本，重新预览并明确接入当前配置", async () => {
  vi.mocked(applyAgentConfig).mockRejectedValueOnce(new Error("agent: config_changed"))
  vi.mocked(rereadAgentConfig).mockResolvedValue({
    ...item,
    revision: "fresh-revision",
    errorCode: "config_changed",
  })
  const user = userEvent.setup()
  mount()
  await user.click(screen.getByRole("button", { name: "配置预览" }))
  await user.click(await screen.findByRole("button", { name: "应用配置" }))
  expect(await screen.findByRole("alert")).toHaveTextContent("重新读取配置")
  await user.click(screen.getByRole("button", { name: "重新读取配置" }))
  expect(await screen.findByRole("status")).toHaveTextContent("旧备份会保留为历史")
  expect(screen.queryByRole("button", { name: "应用配置" })).not.toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "配置预览" }))
  await waitFor(() =>
    expect(previewAgentPlan).toHaveBeenLastCalledWith({
      client: "codex",
      revision: "fresh-revision",
      acceptCurrentConfig: true,
    }),
  )
  await user.click(await screen.findByRole("button", { name: "应用配置" }))
  await waitFor(() =>
    expect(applyAgentConfig).toHaveBeenLastCalledWith({
      client: "codex",
      revision: "fresh-revision",
      catalogRevision: "catalog-revision",
      acceptCurrentConfig: true,
    }),
  )
})
