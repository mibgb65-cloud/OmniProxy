import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import { Dialog } from "radix-ui"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { getApplicationUpdate, type ApplicationUpdate } from "../../services/updates"
import { UpdateDialog } from "./update-dialog"

vi.mock("../../services/updates", () => ({
  getApplicationUpdate: vi.fn(),
  startApplicationUpdate: vi.fn(),
  installApplicationUpdate: vi.fn(),
  cancelApplicationUpdate: vi.fn(),
}))

const notes = [
  "# v2.1.0 - 测试版本说明",
  "",
  "本次正式版汇总本地 `2.0.1-beta.1` 的变更。",
  "",
  "## 新特性",
  "",
  "- **网关端口管理**：保存后重启监听器。",
  "  - **受管客户端**：`OMNIPROXY_PROXY_PORT` 优先。",
  "",
  "## 升级说明",
  "",
  "详见 [升级说明](https://github.com/owner/repo/blob/main/docs/UPGRADE.md)。",
].join("\n")

const state: ApplicationUpdate = {
  currentVersion: "2.0.1-beta.7",
  checkState: "available",
  checkError: "",
  canInstall: true,
  release: {
    version: "v2.1.0",
    notes,
    url: "https://github.com/owner/repo/releases/tag/v2.1.0",
    publishedAt: "2026-10-28T16:03:34Z",
    packageAvailable: true,
    size: 19_922_944,
  },
  download: { state: "idle", version: "", received: 0, total: 0, percent: 0, error: "" },
}

function renderDialog(overrides: Partial<ApplicationUpdate> = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <Dialog.Root open>
          <UpdateDialog state={{ ...state, ...overrides }} />
        </Dialog.Root>
      </QueryClientProvider>
    </I18nProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(getApplicationUpdate).mockResolvedValue(state)
})

it("版本摘要分区展示当前版本、可升级版本与包信息", async () => {
  renderDialog()
  expect(await screen.findByRole("dialog", { name: "升级 OmniProxy" })).toBeInTheDocument()
  const summary = screen.getByRole("region", { name: "版本信息" })
  expect(summary).toHaveTextContent("当前版本")
  expect(summary).toHaveTextContent("2.0.1-beta.7")
  expect(summary).toHaveTextContent("可升级版本")
  expect(summary).toHaveTextContent("v2.1.0")
  expect(summary).toHaveTextContent("19 MB")
  expect(screen.getByRole("link", { name: /打开发布页面/ })).toHaveAttribute(
    "href",
    "https://github.com/owner/repo/releases/tag/v2.1.0",
  )
})

it("更新内容按 Markdown 渲染，不再显示原始记号", async () => {
  const { container } = renderDialog()
  await screen.findByRole("dialog", { name: "升级 OmniProxy" })
  expect(screen.getByRole("heading", { name: "v2.1.0 - 测试版本说明" })).toBeInTheDocument()
  expect(screen.getByRole("heading", { name: "新特性" })).toBeInTheDocument()
  expect(screen.getByText("网关端口管理").tagName).toBe("STRONG")
  expect(screen.getByText("OMNIPROXY_PROXY_PORT").tagName).toBe("CODE")
  expect(screen.getByRole("link", { name: "升级说明" })).toHaveAttribute(
    "href",
    "https://github.com/owner/repo/blob/main/docs/UPGRADE.md",
  )
  const text = container.textContent ?? ""
  expect(text).not.toContain("# v2.1.0")
  expect(text).not.toContain("## 新特性")
  expect(text).not.toContain("**网关端口管理**")
})

it("没有发布说明时给出明确提示而不是空白区域", async () => {
  renderDialog({
    release: { ...state.release!, notes: "   " },
  })
  expect(await screen.findByText("发布者尚未提供更新说明。")).toBeInTheDocument()
})
