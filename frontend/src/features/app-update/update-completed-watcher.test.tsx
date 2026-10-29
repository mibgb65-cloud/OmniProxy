import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi, type Mock } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import type { ApplicationUpdate } from "../../services/updates"
import { useAppPreferences } from "../settings/app-preferences-state"
import { UpdateCompletedWatcher } from "./update-completed-watcher"
import { getApplicationUpdate, acknowledgeApplicationUpdateInstalled } from "../../services/updates"

vi.mock("../../services/updates", () => ({
  getApplicationUpdate: vi.fn(),
  startApplicationUpdate: vi.fn(),
  installApplicationUpdate: vi.fn(),
  cancelApplicationUpdate: vi.fn(),
  acknowledgeApplicationUpdateInstalled: vi.fn(),
}))

const acknowledged: ApplicationUpdate = {
  currentVersion: "2.1.1-beta.2",
  checkState: "idle",
  checkError: "",
  canInstall: true,
  download: {
    state: "installed",
    version: "2.1.1-beta.2",
    received: 0,
    total: 0,
    percent: 100,
    error: "",
  },
}

const installed: ApplicationUpdate = {
  ...acknowledged,
  installed: { version: "2.1.1-beta.2", from: "2.1.1-beta.1" },
}

function renderWatcher() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <UpdateCompletedWatcher />
      </QueryClientProvider>
    </I18nProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  useAppPreferences.setState({ open: false, tab: "settings" })
  ;(acknowledgeApplicationUpdateInstalled as Mock).mockResolvedValue(acknowledged)
})

it("升级后重启时自动弹出一次更新完成提示", async () => {
  ;(getApplicationUpdate as Mock).mockResolvedValue(installed)
  renderWatcher()
  expect(await screen.findByRole("dialog", { name: "更新完成" })).toBeInTheDocument()
  expect(acknowledgeApplicationUpdateInstalled).not.toHaveBeenCalled()
})

it("确认后写入回执并关闭提示", async () => {
  ;(getApplicationUpdate as Mock).mockResolvedValue(installed)
  renderWatcher()
  await userEvent.click(await screen.findByRole("button", { name: "知道了" }))
  await waitFor(() => expect(acknowledgeApplicationUpdateInstalled).toHaveBeenCalledTimes(1))
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
})

it("查看完整更新日志会先回执，再打开设置页的关于分区", async () => {
  ;(getApplicationUpdate as Mock).mockResolvedValue(installed)
  renderWatcher()
  await userEvent.click(await screen.findByRole("button", { name: "查看完整更新日志" }))
  await waitFor(() => expect(acknowledgeApplicationUpdateInstalled).toHaveBeenCalledTimes(1))
  await waitFor(() => expect(useAppPreferences.getState().open).toBe(true))
  expect(useAppPreferences.getState().tab).toBe("about")
})

it("没有待提示的升级时不打扰用户", async () => {
  ;(getApplicationUpdate as Mock).mockResolvedValue(acknowledged)
  renderWatcher()
  await waitFor(() => expect(getApplicationUpdate).toHaveBeenCalled())
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
})
