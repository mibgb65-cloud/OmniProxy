import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { readRequests } from "../../services/activity"
import { useUIStore } from "../../stores/ui-store"
import { RequestsView } from "./requests-view"

vi.mock("../../services/activity", () => ({
  readRequests: vi.fn(),
  readRequest: vi.fn(),
}))
vi.mock("@wailsio/runtime", () => ({ Clipboard: { SetText: vi.fn() } }))

function mount(total: number) {
  vi.mocked(readRequests).mockResolvedValue({ items: [], total, limit: 50, maxRecords: 10000 })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <RequestsView />
      </QueryClientProvider>
    </I18nProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
})

it.each([
  [0, 1],
  [50, 1],
  [51, 2],
  [1017, 21],
  [4351, 88],
])("%i 条请求按每页 50 条显示共 %i 页", async (total, pages) => {
  mount(total)
  await waitFor(() => expect(screen.getByRole("button", { name: "刷新" })).toBeEnabled())
  expect(screen.getByText(`第 1 页 / 共 ${pages} 页`)).toBeInTheDocument()
  expect(screen.getByText(`保留 ${total} 条，上限 10000 条`)).toBeInTheDocument()
  expect(screen.getByRole("button", { name: "上一页" })).toBeDisabled()
  if (pages === 1) {
    expect(screen.getByRole("button", { name: "下一页" })).toBeDisabled()
  }
})

it("不满一页的末页禁用下一页，返回及搜索使用同一个分页大小", async () => {
  const user = userEvent.setup()
  mount(51)
  await screen.findByText("第 1 页 / 共 2 页")
  await user.click(screen.getByRole("button", { name: "下一页" }))
  expect(await screen.findByText("第 2 页 / 共 2 页")).toBeInTheDocument()
  expect(readRequests).toHaveBeenLastCalledWith({ search: "", state: "", offset: 50, limit: 50 })
  expect(screen.getByRole("button", { name: "下一页" })).toBeDisabled()
  await user.click(screen.getByRole("button", { name: "上一页" }))
  expect(await screen.findByText("第 1 页 / 共 2 页")).toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "下一页" }))
  await user.type(
    screen.getByRole("textbox", { name: "搜索请求 ID、模型、Agent、邮箱或账号" }),
    "flash",
  )
  await waitFor(() =>
    expect(readRequests).toHaveBeenLastCalledWith({
      search: "flash",
      state: "",
      offset: 0,
      limit: 50,
    }),
  )
  expect(screen.getByText("第 1 页 / 共 2 页")).toBeInTheDocument()
})

it("英文分页同样显示总页数而非记录数", async () => {
  useUIStore.setState({ language: "en" })
  mount(1017)
  expect(await screen.findByText("Page 1 / 21 pages")).toBeInTheDocument()
})

it("请求页占满内容区可视高度，长列表不把整页顶出窗口", async () => {
  mount(50)
  await waitFor(() => expect(screen.getByRole("button", { name: "刷新" })).toBeEnabled())
  const page = screen.getByRole("heading", { level: 1 }).closest("section")
  expect(page).toHaveClass("app-page-fit")
  expect(page).toHaveClass("flex-col")
  expect(page).not.toHaveClass("space-y-5")
})
