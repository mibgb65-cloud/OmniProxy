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
  ],
  totalBytes: 3072,
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

  it("renders storage locations and sizes", async () => {
    renderOverview()

    await waitFor(() => {
      expect(
        screen.getByText("C:\\Users\\test\\AppData\\Roaming\\OmniProxy-dev"),
      ).toBeInTheDocument()
    })
    // The database row shows "size — path".
    expect(screen.getByText(/^2\.0 KB — /)).toBeInTheDocument()
    expect(screen.getByText("1.0 KB")).toBeInTheDocument() // logs item
    expect(screen.getByText("3.0 KB")).toBeInTheDocument() // total
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
