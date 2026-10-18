import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { previewLegacyImport, importLegacyData } from "../../services/legacy-import"
import { LegacyImportCard } from "./legacy-import-card"

vi.mock("../../services/legacy-import", () => ({
  previewLegacyImport: vi.fn(),
  importLegacyData: vi.fn(),
}))
vi.mock("./use-storage", () => ({
  useSelectDirectory: () => ({ mutate: vi.fn(), isPending: false, error: null }),
}))

const preview = {
  directory: "C:/fake-v1",
  snapshot: "a".repeat(64),
  ready: 1,
  entries: [
    { index: 1, name: "旧账号", provider: "deepseek", state: "ready", enabled: false },
    { index: 2, name: "订阅账号", provider: "openai", state: "login_required", enabled: true },
  ],
}

function renderCard() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <LegacyImportCard />
      </QueryClientProvider>
    </I18nProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(previewLegacyImport).mockResolvedValue(preview)
  vi.mocked(importLegacyData).mockResolvedValue({
    entries: preview.entries.map((entry) => ({
      ...entry,
      state: entry.index === 1 ? "imported" : entry.state,
    })),
    imported: 1,
    failed: 0,
  })
})

it("只在点击后预览并确认导入，展示跳过原因和停用状态", async () => {
  const user = userEvent.setup()
  renderCard()
  expect(previewLegacyImport).not.toHaveBeenCalled()
  expect(importLegacyData).not.toHaveBeenCalled()
  await user.click(screen.getByRole("button", { name: "预览旧版数据" }))
  expect(await screen.findByText("请在新版重新登录订阅")).toBeInTheDocument()
  expect(screen.getByText(/保持停用/)).toBeInTheDocument()
  expect(importLegacyData).not.toHaveBeenCalled()
  await user.click(screen.getByRole("button", { name: "确认导入 1 个账号" }))
  await waitFor(() =>
    expect(importLegacyData).toHaveBeenCalledWith(preview.directory, preview.snapshot),
  )
  expect(await screen.findByText(/导入成功/)).toBeInTheDocument()
  expect(screen.getByRole("button", { name: "确认导入 1 个账号" })).toBeDisabled()
})

it("修改目录清除旧预览，不能提交旧快照", async () => {
  const user = userEvent.setup()
  renderCard()
  await user.click(screen.getByRole("button", { name: "预览旧版数据" }))
  await screen.findByRole("button", { name: "确认导入 1 个账号" })
  await user.type(screen.getByRole("textbox", { name: "旧版数据目录" }), "C:/other")
  expect(screen.queryByRole("button", { name: "确认导入 1 个账号" })).not.toBeInTheDocument()
  expect(importLegacyData).not.toHaveBeenCalled()
})

it("文件变化显示重新预览提示，错误正文不显示在界面", async () => {
  const user = userEvent.setup()
  vi.mocked(importLegacyData).mockRejectedValue(new Error("legacy: source_changed"))
  renderCard()
  await user.click(screen.getByRole("button", { name: "预览旧版数据" }))
  await user.click(await screen.findByRole("button", { name: "确认导入 1 个账号" }))
  expect(await screen.findByRole("alert")).toHaveTextContent(
    "旧版文件在预览后发生变化，请重新预览。",
  )
  expect(screen.queryByText("legacy: source_changed")).not.toBeInTheDocument()
})
