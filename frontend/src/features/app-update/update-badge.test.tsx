import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import {
  getApplicationUpdate,
  startApplicationUpdate,
  installApplicationUpdate,
  cancelApplicationUpdate,
  type ApplicationUpdate,
} from "../../services/updates"
import { UpdateBadge } from "./update-badge"

vi.mock("../../services/updates", () => ({
  getApplicationUpdate: vi.fn(),
  startApplicationUpdate: vi.fn(),
  installApplicationUpdate: vi.fn(),
  cancelApplicationUpdate: vi.fn(),
}))

const available: ApplicationUpdate = {
  currentVersion: "2.0.0-beta.1",
  checkState: "available",
  checkError: "",
  canInstall: true,
  release: {
    version: "v2.0.0-beta.2",
    notes: "修复升级流程",
    url: "https://github.com/owner/repo/releases/tag/v2.0.0-beta.2",
    publishedAt: "",
    packageAvailable: true,
    size: 1000,
  },
  download: { state: "idle", version: "", received: 0, total: 0, percent: 0, error: "" },
}

function renderBadge(collapsed = false) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const onOpen = vi.fn()
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <UpdateBadge collapsed={collapsed} onOpen={onOpen} />
      </QueryClientProvider>
    </I18nProvider>,
  )
  return onOpen
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(getApplicationUpdate).mockResolvedValue(available)
})

it("无新版本时隐藏标记，不自动下载", async () => {
  vi.mocked(getApplicationUpdate).mockResolvedValue({
    ...available,
    checkState: "up-to-date",
    release: null,
  })
  renderBadge()
  await waitFor(() => expect(getApplicationUpdate).toHaveBeenCalled())
  expect(screen.queryByRole("button")).not.toBeInTheDocument()
  expect(startApplicationUpdate).not.toHaveBeenCalled()
})

it.each([false, true])("显示标记，点击后展示说明并下载、重启（折叠=%s）", async (collapsed) => {
  const user = userEvent.setup()
  const ready = {
    ...available,
    download: {
      ...available.download,
      state: "ready",
      version: "v2.0.0-beta.2",
      percent: 100,
      received: 1000,
      total: 1000,
    },
  }
  vi.mocked(startApplicationUpdate).mockResolvedValue(ready)
  vi.mocked(installApplicationUpdate).mockResolvedValue({
    ...ready,
    download: { ...ready.download, state: "installing" },
  })
  const onOpen = renderBadge(collapsed)
  await user.click(
    await screen.findByRole("button", { name: "发现新版本 v2.0.0-beta.2，查看升级" }),
  )
  expect(onOpen).toHaveBeenCalledTimes(1)
  expect(await screen.findByRole("dialog", { name: "升级 OmniProxy" })).toBeInTheDocument()
  expect(screen.getByText("修复升级流程")).toBeInTheDocument()
  expect(startApplicationUpdate).not.toHaveBeenCalled()
  await user.click(screen.getByRole("button", { name: "下载更新包" }))
  await waitFor(() => expect(startApplicationUpdate).toHaveBeenCalledWith("v2.0.0-beta.2"))
  await user.click(await screen.findByRole("button", { name: "重启并静默升级" }))
  await waitFor(() => expect(installApplicationUpdate).toHaveBeenCalledWith("v2.0.0-beta.2"))
  expect(await screen.findByText("正在准备安装，完成后将自动重新打开…")).toBeInTheDocument()
})

it("下载显示共享进度并可取消，校验失败不提供安装", async () => {
  const user = userEvent.setup()
  vi.mocked(getApplicationUpdate).mockResolvedValue({
    ...available,
    download: {
      state: "downloading",
      version: "v2.0.0-beta.2",
      received: 400,
      total: 1000,
      percent: 40,
      error: "",
    },
  })
  vi.mocked(cancelApplicationUpdate).mockResolvedValue({
    ...available,
    download: {
      state: "failed",
      version: "v2.0.0-beta.2",
      received: 400,
      total: 1000,
      percent: 40,
      error: "update: checksum_mismatch",
    },
  })
  renderBadge()
  await user.click(
    await screen.findByRole("button", { name: "发现新版本 v2.0.0-beta.2，查看升级" }),
  )
  expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "40")
  await user.click(screen.getByRole("button", { name: "取消下载" }))
  expect(await screen.findByRole("alert")).toHaveTextContent("安装包校验失败")
  expect(screen.queryByRole("button", { name: "重启并静默升级" })).not.toBeInTheDocument()
  expect(installApplicationUpdate).not.toHaveBeenCalled()
})
