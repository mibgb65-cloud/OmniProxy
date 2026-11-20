import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { readRecordedRequests, readRetention } from "../../services/activity"
import { useUIStore } from "../../stores/ui-store"
import { RecordsView } from "./records-view"

vi.mock("../../services/activity", () => ({
  readRecordedRequests: vi.fn(),
  readRetention: vi.fn(),
}))

const item = {
  id: "a".repeat(32),
  startedAt: "2026-11-20T00:19:59Z",
  capturedAt: "2026-11-20T00:19:59Z",
  model: "deepseek-v4",
  providerName: "River",
  accountName: "Member",
  client: "codex",
  state: "success",
  protocol: "chat",
  transport: "http",
  preview: "你好，帮我检查这个错误",
  originalBytes: 300000,
  storedBytes: 262144,
  truncated: true,
}

function mount(onOpenRequest = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <RecordsView onOpenRequest={onOpenRequest} />
      </QueryClientProvider>
    </I18nProvider>,
  )
  return onOpenRequest
}

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(readRetention).mockResolvedValue({
    maxRecords: 10000,
    storedRecords: 4,
    logCapacity: 5000,
    contentEnabled: true,
    contentBytes: 0,
    contentBudget: 536870912,
  })
})

it("列出已保存正文的记录并在关闭时提示历史内容", async () => {
  vi.mocked(readRetention).mockResolvedValue({
    maxRecords: 10000,
    storedRecords: 4,
    logCapacity: 5000,
    contentEnabled: false,
    contentBytes: 0,
    contentBudget: 536870912,
  })
  vi.mocked(readRecordedRequests).mockResolvedValue({ items: [item], total: 1, limit: 20 })
  const open = mount()
  expect(await screen.findByText("正文记录已关闭")).toBeInTheDocument()
  expect(screen.getByText("你好，帮我检查这个错误")).toBeInTheDocument()
  expect(screen.getByText("已截断")).toBeInTheDocument()
  expect(screen.getByText("共 1 条已保存正文")).toBeInTheDocument()
  await userEvent.setup().click(screen.getByRole("button", { name: /查看记录/ }))
  expect(open).toHaveBeenCalledWith(item.id)
})

it("开启记录但没有记录时说明等待新请求", async () => {
  vi.mocked(readRecordedRequests).mockResolvedValue({ items: [], total: 0, limit: 20 })
  mount()
  expect(await screen.findByText("已开启记录，等待新请求")).toBeInTheDocument()
})

it("搜索重置分页并按关键字查询", async () => {
  vi.mocked(readRecordedRequests).mockResolvedValue({ items: [], total: 45, limit: 20 })
  mount()
  const user = userEvent.setup()
  expect(await screen.findByText("第 1 页 / 共 3 页")).toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "下一页" }))
  await waitFor(() =>
    expect(readRecordedRequests).toHaveBeenLastCalledWith({ search: "", offset: 20, limit: 20 }),
  )
  await user.type(screen.getByRole("textbox", { name: "搜索请求 ID、模型、Agent 或账号" }), "gpt")
  await waitFor(() =>
    expect(readRecordedRequests).toHaveBeenLastCalledWith({ search: "gpt", offset: 0, limit: 20 }),
  )
})
