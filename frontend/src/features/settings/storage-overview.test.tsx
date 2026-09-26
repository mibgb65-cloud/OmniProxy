import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { cleanStorage, getStorageInfo, type StorageInfo } from "../../services/omniproxy"
import { StorageOverview } from "./storage-overview"

vi.mock("../../services/omniproxy", () => ({
  getStorageInfo: vi.fn(),
  setDataDir: vi.fn(),
  selectDirectory: vi.fn(),
  cleanStorage: vi.fn(),
}))

const storageInfo: StorageInfo = {
  dataDir: "C:\\Users\\test\\AppData\\Roaming\\OmniProxy-dev",
  logDir: "C:\\Users\\test\\AppData\\Roaming\\OmniProxy-dev\\logs",
  databasePath: "C:\\Users\\test\\AppData\\Roaming\\OmniProxy-dev\\omniproxy.db",
  pendingDataDir: "",
  items: [
    { name: "database", path: "…/omniproxy.db", bytes: 2048 },
    { name: "logs", path: "…/logs", bytes: 1024 },
    { name: "cache", path: "…/cache", bytes: 512 },
    { name: "other", path: "…", bytes: 512 },
  ],
  totalBytes: 4096,
}

function renderOverview() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <I18nProvider>
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      </I18nProvider>
    )
  }
  return render(<StorageOverview />, { wrapper: Wrapper })
}

describe("StorageOverview", () => {
  beforeEach(() => {
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    vi.mocked(getStorageInfo).mockResolvedValue(storageInfo)
  })

  it("renders storage locations and an accessible usage breakdown", async () => {
    renderOverview()

    await waitFor(() => {
      expect(
        screen.getByText("C:\\Users\\test\\AppData\\Roaming\\OmniProxy-dev"),
      ).toBeInTheDocument()
    })
    expect(screen.getByRole("img", { name: "数据存储占用比例" })).toBeInTheDocument()
    expect(screen.getByText("数据库文件")).toBeInTheDocument()
    expect(screen.getByText("2.0 KB")).toBeInTheDocument()
    expect(screen.getByText("1.0 KB")).toBeInTheDocument()
    expect(screen.getAllByText("512 B")).toHaveLength(2)
    expect(screen.getByText("4.0 KB")).toBeInTheDocument()
    expect(screen.getByText("50%")).toBeInTheDocument()
    expect(
      screen.getByText("账号元数据计入数据库，无法按文件大小单独拆分；系统凭据不在数据目录中。"),
    ).toBeInTheDocument()
  })

  it("shows zero usage without dividing by zero", async () => {
    vi.mocked(getStorageInfo).mockResolvedValue({ ...storageInfo, items: [], totalBytes: 0 })
    renderOverview()

    expect(await screen.findByText("占用分布")).toBeInTheDocument()
    expect(screen.getAllByText("0 B")).toHaveLength(5)
    expect(screen.getAllByText("0%")).toHaveLength(4)
  })

  it("refreshes the storage snapshot on request", async () => {
    const user = userEvent.setup()
    renderOverview()

    await screen.findByText("占用分布")
    await user.click(screen.getByRole("button", { name: "刷新" }))

    await waitFor(() => expect(getStorageInfo).toHaveBeenCalledTimes(2))
  })

  it("cleans logs and reports freed space", async () => {
    const user = userEvent.setup()
    vi.mocked(cleanStorage).mockResolvedValue({ target: "logs", freedBytes: 512 })
    renderOverview()

    await user.click(await screen.findByRole("button", { name: "清理日志" }))

    await waitFor(() => {
      expect(cleanStorage).toHaveBeenCalledWith("logs")
    })
    expect(await screen.findByText("已释放 512 B。")).toBeInTheDocument()
  })

  it("compacts the database", async () => {
    const user = userEvent.setup()
    vi.mocked(cleanStorage).mockResolvedValue({ target: "database", freedBytes: 0 })
    renderOverview()

    await user.click(await screen.findByRole("button", { name: "压缩数据库" }))

    await waitFor(() => {
      expect(cleanStorage).toHaveBeenCalledWith("database")
    })
  })

  it("shows a pending location when one is stored", async () => {
    vi.mocked(getStorageInfo).mockResolvedValue({
      ...storageInfo,
      pendingDataDir: "D:\\omniproxy-data",
    })
    renderOverview()

    expect(await screen.findByText("待生效的新位置：D:\\omniproxy-data")).toBeInTheDocument()
  })
})
