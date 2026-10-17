import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { applyAgentConfig, previewAgentPlan } from "../../services/agents"
import { useUIStore } from "../../stores/ui-store"
import { AgentConfigEditor } from "./agent-config-editor"

vi.mock("../../services/agents", () => ({
  previewAgentPlan: vi.fn(),
  applyAgentConfig: vi.fn(),
  rereadAgentConfig: vi.fn(),
  agentError: () => "catalog_changed",
}))
beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(previewAgentPlan).mockResolvedValue({
    summary: "",
    mode: "cli-model-list",
    defaultModel: "a",
    catalogRevision: "catalog",
    models: ["a", "b", "a[1m]", "b[1m]"].map((id) => ({
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
        <AgentConfigEditor
          item={{
            client: "claude",
            path: "test/settings.json",
            exists: true,
            configured: false,
            provider: "",
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

it("配置使用整卡勾选，快筛隐藏的选择和默认模型保留，提交与实际勾选一致", async () => {
  const user = await open()
  const filter = screen.getByRole("button", { name: "仅看 [1m]" })
  await user.click(filter)
  expect(filter).toHaveAttribute("aria-pressed", "true")
  expect(screen.getAllByRole("checkbox")).toHaveLength(2)
  expect(screen.queryByRole("checkbox", { name: "配置 Model a" })).not.toBeInTheDocument()
  const card = screen.getByRole("checkbox", { name: "配置 Model a[1m]" }).closest("label")!
  await user.click(card)
  expect(screen.getByRole("checkbox", { name: "配置 Model a[1m]" })).not.toBeChecked()
  expect(screen.getByText("已选 3 / 4 个")).toBeInTheDocument()
  expect(screen.getByRole("combobox", { name: "默认模型" })).toHaveTextContent("Model a")
  await user.click(filter)
  expect(screen.getByRole("checkbox", { name: "配置 Model a" })).toBeChecked()
  expect(screen.getByRole("checkbox", { name: "配置 Model b" })).toBeChecked()
  expect(applyAgentConfig).not.toHaveBeenCalled()
  await user.click(screen.getByRole("button", { name: "应用配置" }))
  await waitFor(() =>
    expect(applyAgentConfig).toHaveBeenCalledWith({
      client: "claude",
      revision: "config",
      catalogRevision: "catalog",
      modelIds: ["a", "b", "b[1m]"],
      model: "a",
    }),
  )
})

it("快筛叠加搜索只改变展示，全选仍选全部，卡片支持空格切换", async () => {
  const user = await open()
  await user.click(screen.getByRole("button", { name: "清空选择" }))
  await user.click(screen.getByRole("button", { name: "仅看 [1m]" }))
  await user.type(screen.getByRole("searchbox"), "a[1m]")
  expect(screen.getAllByRole("checkbox")).toHaveLength(1)
  expect(screen.getByRole("button", { name: "应用配置" })).toBeDisabled()
  await user.click(screen.getByRole("button", { name: "全选" }))
  expect(screen.getByText("已选 4 / 4 个")).toBeInTheDocument()
  expect(screen.getByText("显示 1 / 4 个")).toBeInTheDocument()
  const checkbox = screen.getByRole("checkbox", { name: "配置 Model a[1m]" })
  checkbox.focus()
  await user.keyboard(" ")
  expect(checkbox).not.toBeChecked()
  await user.keyboard(" ")
  expect(checkbox).toBeChecked()
  await user.click(screen.getByRole("button", { name: "应用配置" }))
  await waitFor(() =>
    expect(applyAgentConfig).toHaveBeenCalledWith({
      client: "claude",
      revision: "config",
      catalogRevision: "catalog",
    }),
  )
})
