import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, within } from "@testing-library/react"
import { beforeEach, expect, it, vi } from "vitest"
import type { RequestItem } from "../../services/activity"
import { readRequests } from "../../services/activity"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { RequestTable } from "./request-table"

vi.mock("../../services/activity", async (original) => ({
  ...(await original<typeof import("../../services/activity")>()),
  readRequests: vi.fn(),
}))
vi.mock("@wailsio/runtime", () => ({ Clipboard: { SetText: vi.fn() } }))

const item: RequestItem = {
  id: "a".repeat(32),
  startedAt: "2026-10-23T01:02:03Z",
  finishedAt: "2026-10-23T01:02:09Z",
  protocol: "responses",
  transport: "http",
  inputModel: "gpt-6.1-sol",
  model: "gpt-6.1-sol",
  stream: true,
  providerId: "builtin_openai",
  providerName: "OpenAI",
  accountId: "acct",
  accountName: "member",
  email: "member@example.invalid",
  workspaceId: "",
  accountDeleted: false,
  state: "success",
  statusCode: 200,
  errorCode: "",
  latencyMs: 4200,
  inputTokens: "1200",
  outputTokens: "300",
  inputKnown: true,
  outputKnown: true,
  usageComplete: true,
}

function mount(items: RequestItem[]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  vi.mocked(readRequests).mockResolvedValue({
    items,
    total: items.length,
    limit: 50,
    maxRecords: 10000,
  })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <RequestTable items={items} onSelect={() => {}} />
      </QueryClientProvider>
    </I18nProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
})

it("按接入 Key 归属显示发起请求的 Agent", async () => {
  mount([
    { ...item, client: "codex" },
    { ...item, id: "b".repeat(32), client: "harness-web", email: "other@example.invalid" },
  ])
  const rows = screen.getAllByRole("row").slice(1)
  expect(within(rows[0]!).getByText("Codex CLI / Desktop")).toBeInTheDocument()
  expect(within(rows[1]!).getByText("DeepSeek Harness Web")).toBeInTheDocument()
})

it("无法归属的请求显示破折号而不是猜测对应 Agent", async () => {
  mount([{ ...item }, { ...item, id: "c".repeat(32), client: "unknown-client" }])
  const rows = screen.getAllByRole("row").slice(1)
  expect(within(rows[0]!).getByText("—")).toBeInTheDocument()
  expect(within(rows[1]!).getByText("—")).toBeInTheDocument()
})
