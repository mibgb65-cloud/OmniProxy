import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor, within } from "@testing-library/react"
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

it.each(["zh", "en"] as const)(
  "%s 快速筛选精确的 [1m] 后缀，可叠加搜索并再次取消",
  async (language) => {
    useUIStore.setState({ language })
    vi.mocked(getConfiguredAgentModels).mockResolvedValue({
      models: [
        { id: "alpha", name: "Alpha" },
        { id: "alpha[1m]", name: "Alpha Long" },
        { id: "beta[1m]", name: "Beta Long" },
        { id: "contains-[1m]-text", name: "Suffix Lookalike" },
      ],
      defaultModel: "alpha[1m]",
    })
    const user = userEvent.setup()
    mount()
    await screen.findByText("Alpha Long")
    const filter = screen.getByRole("button", {
      name: language === "zh" ? "仅看 [1m]" : "[1m] only",
    })
    const list = screen.getByRole("list")
    expect(within(list).getAllByRole("listitem")).toHaveLength(4)
    expect(filter).toHaveAttribute("aria-pressed", "false")
    await user.click(filter)
    expect(filter).toHaveAttribute("aria-pressed", "true")
    expect(within(list).getAllByRole("listitem")).toHaveLength(2)
    expect(within(list).queryByText("Suffix Lookalike")).not.toBeInTheDocument()
    expect(
      within(list).getByText(language === "zh" ? "默认模型" : "Default model"),
    ).toBeInTheDocument()
    await user.type(screen.getByRole("searchbox"), "ALPHA")
    expect(within(list).getAllByRole("listitem")).toHaveLength(1)
    expect(within(list).getByText("Alpha Long")).toBeInTheDocument()
    await user.click(filter)
    expect(within(list).getAllByRole("listitem")).toHaveLength(2)
    expect(screen.getByRole("searchbox")).toHaveValue("ALPHA")
    expect(getConfiguredAgentModels).toHaveBeenCalledTimes(1)
  },
)

it("没有 [1m] 时显示空匹配状态，键盘可取消筛选恢复全部", async () => {
  const user = userEvent.setup()
  mount()
  await screen.findByText("Saved Model")
  const filter = screen.getByRole("button", { name: "仅看 [1m]" })
  filter.focus()
  await user.keyboard(" ")
  expect(filter).toHaveAttribute("aria-pressed", "true")
  expect(screen.getByText("没有匹配的模型。")).toBeInTheDocument()
  expect(screen.getByText("显示 0 / 1 个")).toBeInTheDocument()
  await user.keyboard(" ")
  expect(screen.getByText("Saved Model")).toBeInTheDocument()
})
