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
  agentError: () => "failed",
}))

const entry = (id: string, efforts?: string[]) => ({
  id,
  name: `Model ${id}`,
  targetModel: id,
  providerId: "river",
  providerKind: "river",
  alias: false,
  metadata: efforts
    ? {
        reasoningEfforts: efforts.map((value) => ({ value, description: `level ${value}` })),
        defaultEffort: efforts[0],
      }
    : {},
})

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(previewAgentPlan).mockResolvedValue({
    summary: "",
    mode: "cli-model-list",
    defaultModel: "river-effort",
    catalogRevision: "catalog",
    models: [entry("river-effort", ["high", "max"]), entry("plain-model")],
    efforts: { "river-effort": "high" },
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
  await screen.findByRole("checkbox", { name: "配置 Model river-effort" })
  return user
}

it("已确认档位的模型才有强度选择，并显示已保存的档位", async () => {
  await open()
  const control = screen.getByRole("combobox", { name: "Model river-effort 的思考强度" })
  // 已保存档位来自目录数据；未确认档位的模型不出现选择入口。
  expect(control).toHaveTextContent("level high")
  expect(
    screen.queryByRole("combobox", { name: "Model plain-model 的思考强度" }),
  ).not.toBeInTheDocument()
})

it("选择档位后随应用一起提交，回到跟随客户端则提交空值", async () => {
  const user = await open()
  const control = screen.getByRole("combobox", { name: "Model river-effort 的思考强度" })
  await user.click(control)
  await user.click(screen.getByRole("option", { name: "level max（max）" }))
  await user.click(screen.getByRole("button", { name: "应用配置" }))
  await waitFor(() =>
    expect(applyAgentConfig).toHaveBeenCalledWith(
      expect.objectContaining({ client: "claude", efforts: { "river-effort": "max" } }),
    ),
  )
})

it("把档位改回已保存值时不提交额外字段", async () => {
  const user = await open()
  const control = screen.getByRole("combobox", { name: "Model river-effort 的思考强度" })
  await user.click(control)
  await user.click(screen.getByRole("option", { name: "level max（max）" }))
  await user.click(screen.getByRole("combobox", { name: "Model river-effort 的思考强度" }))
  await user.click(screen.getByRole("option", { name: "level high（high）" }))
  await user.click(screen.getByRole("button", { name: "应用配置" }))
  await waitFor(() =>
    expect(applyAgentConfig).toHaveBeenCalledWith(
      expect.not.objectContaining({ efforts: expect.anything() }),
    ),
  )
})
