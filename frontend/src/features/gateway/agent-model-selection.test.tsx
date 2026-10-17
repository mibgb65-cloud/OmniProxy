import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { applyAgentConfig, previewAgentPlan } from "../../services/agents"
import { useUIStore } from "../../stores/ui-store"
import { AgentConfigDialog } from "./agent-config-dialog"

vi.mock("../../services/agents", () => ({
  previewAgentPlan: vi.fn(),
  applyAgentConfig: vi.fn(),
  rereadAgentConfig: vi.fn(),
  agentError: () => "catalog_changed",
  agentTitle: () => "agents.codex",
}))

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(previewAgentPlan).mockResolvedValue({
    summary: "",
    mode: "remote-catalog",
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
  vi.mocked(applyAgentConfig).mockResolvedValue([])
})

async function open() {
  const user = userEvent.setup()
  render(
    <I18nProvider>
      <QueryClientProvider client={new QueryClient()}>
        <AgentConfigDialog
          item={{
            client: "codex",
            path: "test/config.toml",
            exists: true,
            configured: false,
            provider: "custom",
            canRestore: false,
            revision: "config",
            errorCode: "",
          }}
          onClose={vi.fn()}
        />
      </QueryClientProvider>
    </I18nProvider>,
  )
  await user.click(screen.getByRole("button", { name: "配置预览" }))
  await screen.findByRole("checkbox", { name: "配置 Model a" })
  return user
}

it("默认全选；取消原默认后，预览和写入仅保留所选模型并切换默认", async () => {
  const user = await open()
  expect(screen.getByRole("checkbox", { name: "配置 Model a" })).toBeChecked()
  expect(screen.getByRole("checkbox", { name: "配置 Model b" })).toBeChecked()
  await user.click(screen.getByRole("checkbox", { name: "配置 Model a" }))
  expect(screen.getByText("将提供 1 个模型／路由")).toBeInTheDocument()
  expect(screen.getByText("应用管理的本地目录")).toBeInTheDocument()
  expect(screen.getByRole("combobox", { name: "默认模型" })).toHaveTextContent("Model b")
  await user.click(screen.getByRole("button", { name: "应用配置" }))
  await waitFor(() =>
    expect(applyAgentConfig).toHaveBeenCalledWith({
      client: "codex",
      revision: "config",
      catalogRevision: "catalog",
      modelIds: ["b"],
      model: "b",
    }),
  )
})

it("清空会阻止应用；搜索只筛选展示，全选恢复动态全部模型", async () => {
  const user = await open()
  await user.click(screen.getByRole("button", { name: "清空选择" }))
  expect(screen.getByRole("alert")).toHaveTextContent("请至少选择一个模型")
  expect(screen.getByRole("button", { name: "应用配置" })).toBeDisabled()
  await user.type(screen.getByRole("searchbox"), "Model b")
  expect(screen.queryByRole("checkbox", { name: "配置 Model a" })).not.toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "全选" }))
  expect(screen.getByText("已选 2 / 2 个")).toBeInTheDocument()
  expect(screen.getByRole("checkbox", { name: "配置 Model b" })).toBeChecked()
  await user.click(screen.getByRole("button", { name: "应用配置" }))
  await waitFor(() =>
    expect(applyAgentConfig).toHaveBeenCalledWith({
      client: "codex",
      revision: "config",
      catalogRevision: "catalog",
    }),
  )
})

it("全选时也可以单独指定默认模型", async () => {
  const user = await open()
  await user.click(screen.getByRole("combobox", { name: "默认模型" }))
  await user.click(screen.getByRole("option", { name: "Model b" }))
  await user.click(screen.getByRole("button", { name: "应用配置" }))
  await waitFor(() =>
    expect(applyAgentConfig).toHaveBeenCalledWith({
      client: "codex",
      revision: "config",
      catalogRevision: "catalog",
      model: "b",
    }),
  )
})
