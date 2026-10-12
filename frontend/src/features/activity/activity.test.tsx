import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import {
  readRequest,
  readRequests,
  readUsage,
  formatTokens,
  type RequestItem,
} from "../../services/activity"
import { UsageView } from "./usage-view"
import { RequestsView } from "./requests-view"
import { useUIStore } from "../../stores/ui-store"
vi.mock("../../services/activity", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../services/activity")>()),
  readUsage: vi.fn(),
  readRequests: vi.fn(),
  readRequest: vi.fn(),
}))
vi.mock("@wailsio/runtime", () => ({ Clipboard: { SetText: vi.fn() } }))
function mount(element: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>{element}</QueryClientProvider>
    </I18nProvider>,
  )
}
const item: RequestItem = {
  id: "a".repeat(32),
  startedAt: "2026-10-09T03:00:00Z",
  finishedAt: "2026-10-09T03:00:02Z",
  protocol: "responses",
  transport: "http",
  inputModel: "coding",
  model: "gpt-test",
  stream: false,
  providerId: "openai",
  providerName: "OpenAI",
  accountId: "deleted",
  accountName: "Old account",
  email: "old@example.invalid",
  workspaceId: "shared-team",
  accountDeleted: true,
  state: "success",
  statusCode: 200,
  errorCode: "",
  latencyMs: 2000,
  inputTokens: "0",
  outputTokens: "3",
  inputKnown: false,
  outputKnown: true,
  usageComplete: false,
}
beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
})
it("大整数 Token 保持精确，累计统计和报告覆盖率可见", async () => {
  expect(formatTokens("9007199254740993")).toBe("9,007,199,254,740,993")
  vi.mocked(readUsage).mockResolvedValue({
    allTime: {
      inputTokens: "9007199254740993",
      outputTokens: "7",
      totalTokens: "9007199254741000",
      requests: 2,
      completeUsage: 1,
    },
    days: [],
    accounts: [],
    models: [],
  })
  mount(<UsageView />)
  expect(await screen.findByText("9,007,199,254,741,000")).toBeInTheDocument()
  expect(screen.getByText("完整用量 1 / 2 请求")).toBeInTheDocument()
})
it("已删除账号和部分用量保持可见，详情按 ID 跳到关联日志", async () => {
  vi.mocked(readRequests).mockResolvedValue({
    items: [item],
    total: 1,
    limit: 50,
    maxRecords: 10000,
  })
  vi.mocked(readRequest).mockResolvedValue(item)
  const open = vi.fn()
  const user = userEvent.setup()
  mount(<RequestsView onOpenLogs={open} />)
  expect(await screen.findByText("账号已删除")).toBeInTheDocument()
  expect(screen.getByText("部分用量")).toBeInTheDocument()
  expect(screen.getAllByRole("cell").some((cell) => cell.textContent === "—")).toBe(true)
  await user.click(screen.getByRole("button", { name: `查看请求 ${item.id}` }))
  expect(await screen.findByRole("dialog")).toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "关联日志" }))
  expect(open).toHaveBeenCalledWith(item.id)
})
