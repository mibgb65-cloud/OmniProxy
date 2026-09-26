import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import {
  getStorageInfo,
  selectDirectory,
  setDataDir,
  type StorageInfo,
} from "../../services/omniproxy"
import { DataDirForm } from "./data-dir-form"

vi.mock("../../services/omniproxy", () => ({
  getStorageInfo: vi.fn(),
  setDataDir: vi.fn(),
  selectDirectory: vi.fn(),
  cleanStorage: vi.fn(),
}))

const storageInfo: StorageInfo = {
  dataDir: "C:\\data",
  logDir: "C:\\data\\logs",
  databasePath: "C:\\data\\omniproxy.db",
  pendingDataDir: "",
  items: [
    { name: "database", path: "C:\\data\\omniproxy.db", bytes: 1024 },
    { name: "logs", path: "C:\\data\\logs", bytes: 512 },
  ],
  totalBytes: 1536,
}

function renderForm() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <I18nProvider>
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      </I18nProvider>
    )
  }
  return render(<DataDirForm />, { wrapper: Wrapper })
}

describe("DataDirForm", () => {
  beforeEach(() => {
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    vi.mocked(getStorageInfo).mockResolvedValue(storageInfo)
  })

  it("requires a path before submitting", async () => {
    const user = userEvent.setup()
    renderForm()

    expect(screen.getByText(/迁移时间取决于数据量和磁盘速度/)).toBeInTheDocument()
    expect(screen.getByText(/退出应用时自动迁移现有数据/)).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "保存" }))

    expect(await screen.findByText("请输入目录路径。")).toBeInTheDocument()
    expect(setDataDir).not.toHaveBeenCalled()
  })

  it("submits the entered path and shows the restart notice", async () => {
    const user = userEvent.setup()
    vi.mocked(setDataDir).mockResolvedValue({
      previousDir: "C:\\data",
      newDir: "D:\\omniproxy-data",
      requiresRestart: true,
      existingDataSize: 1536,
    })
    renderForm()

    await user.type(screen.getByLabelText("新数据目录"), "D:\\omniproxy-data")
    await user.click(screen.getByRole("button", { name: "保存" }))

    await waitFor(() => {
      expect(setDataDir).toHaveBeenCalledWith("D:\\omniproxy-data")
    })
    expect(await screen.findByText(/已保存。重启后生效/)).toBeInTheDocument()
  })

  it("fills the path from the native folder picker", async () => {
    const user = userEvent.setup()
    vi.mocked(selectDirectory).mockResolvedValue("D:\\picked")
    renderForm()

    await user.click(screen.getByRole("button", { name: "浏览…" }))

    await waitFor(() => {
      expect(selectDirectory).toHaveBeenCalledWith("选择数据目录")
    })
    await waitFor(() => {
      expect(screen.getByLabelText<HTMLInputElement>("新数据目录").value).toBe("D:\\picked")
    })
  })

  it("does nothing when the folder picker is cancelled", async () => {
    const user = userEvent.setup()
    vi.mocked(selectDirectory).mockResolvedValue("")
    renderForm()

    await user.click(screen.getByRole("button", { name: "浏览…" }))

    await waitFor(() => {
      expect(selectDirectory).toHaveBeenCalled()
    })
    expect(screen.getByLabelText<HTMLInputElement>("新数据目录").value).toBe("")
  })

  it("shows an error when the folder picker fails", async () => {
    const user = userEvent.setup()
    vi.mocked(selectDirectory).mockRejectedValue(new Error("picker unavailable"))
    renderForm()

    await user.click(screen.getByRole("button", { name: "浏览…" }))

    expect(await screen.findByRole("alert")).toHaveTextContent("选择目录失败：picker unavailable")
  })

  it("surfaces backend validation errors", async () => {
    const user = userEvent.setup()
    vi.mocked(setDataDir).mockRejectedValue(new Error("profile: D:\\x is not writable"))
    renderForm()

    await user.type(screen.getByLabelText("新数据目录"), "D:\\x")
    await user.click(screen.getByRole("button", { name: "保存" }))

    expect(await screen.findByText("profile: D:\\x is not writable")).toBeInTheDocument()
  })
})
