import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { getConfiguredAgentModels } from "../../services/agents"
import { useUIStore } from "../../stores/ui-store"
import { AgentConnectedModels } from "./agent-connected-models"

vi.mock("../../services/agents", () => ({
  getConfiguredAgentModels: vi.fn(),
  agentError: () => "config_changed",
}))
beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getConfiguredAgentModels).mockResolvedValue({
    models: [{ id: "saved", name: "Saved Model" }],
    defaultModel: "saved",
  })
})
function mount(configured = true) {
  render(
    <I18nProvider>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <AgentConnectedModels
          item={{
            client: "codex",
            path: "test/config.toml",
            exists: true,
            configured,
            provider: "openai",
            canRestore: true,
            revision: "saved",
            errorCode: "",
          }}
        />
      </QueryClientProvider>
    </I18nProvider>,
  )
}
it("搜索只筛选已接入列表，停用 Key 后仍展示模型及明确状态", async () => {
  const user = userEvent.setup()
  mount(false)
  expect(await screen.findByText("Saved Model")).toBeInTheDocument()
  expect(screen.getByText("接入 Key 已停用，模型配置仍保留。")).toBeInTheDocument()
  expect(screen.getByText("默认模型")).toBeInTheDocument()
  await user.type(screen.getByRole("searchbox"), "not-saved")
  expect(screen.getByText("没有匹配的模型。")).toBeInTheDocument()
  expect(getConfiguredAgentModels).toHaveBeenCalledTimes(1)
})
it("外部修改显示安全错误，刷新成功后显示实际目录", async () => {
  vi.mocked(getConfiguredAgentModels).mockRejectedValueOnce(new Error("private raw config"))
  const user = userEvent.setup()
  mount()
  expect(await screen.findByRole("alert")).toHaveTextContent("重新读取配置")
  expect(screen.queryByText("private raw config")).not.toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "刷新模型" }))
  expect(await screen.findByText("Saved Model")).toBeInTheDocument()
  await waitFor(() => expect(screen.queryByRole("alert")).not.toBeInTheDocument())
})
