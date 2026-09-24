import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { checkForUpdate, getRuntimeInfo, type RuntimeInfo } from "../../services/omniproxy"
import { AboutView } from "./about-view"

vi.mock("../../services/omniproxy", () => ({
  getAppInfo: vi.fn(),
  ping: vi.fn(),
  getStorageInfo: vi.fn(),
  setDataDir: vi.fn(),
  selectDirectory: vi.fn(),
  cleanStorage: vi.fn(),
  getRuntimeInfo: vi.fn(),
  checkForUpdate: vi.fn(),
}))

const runtimeInfo: RuntimeInfo = {
  version: "0.1.0",
  appName: "OmniProxy",
  description: "Local AI API gateway",
  appId: "com.omniproxy.app.test",
  profile: "test",
  dataDir: "/tmp/omniproxy-test",
  logDir: "/tmp/omniproxy-test/logs",
  proxyPort: 3002,
  controlPort: 3892,
  goVersion: "go1.26.2",
  wailsVersion: "v3.0.0-beta.25",
  platform: "windows/amd64",
  sqliteVersion: "3.50.4",
  startedAt: new Date(Date.now() - 65_000).toISOString(),
}

function renderAbout() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <I18nProvider>
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      </I18nProvider>
    )
  }
  return render(<AboutView />, { wrapper: Wrapper })
}

describe("AboutView", () => {
  beforeEach(() => {
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    vi.mocked(getRuntimeInfo).mockResolvedValue(runtimeInfo)
  })

  it("renders runtime details", async () => {
    renderAbout()

    await waitFor(() => {
      expect(screen.getByText("go1.26.2")).toBeInTheDocument()
    })
    expect(screen.getByText("v3.0.0-beta.25")).toBeInTheDocument()
    expect(screen.getByText("windows/amd64")).toBeInTheDocument()
    expect(screen.getByText("3.50.4")).toBeInTheDocument()
    expect(screen.getByText("3002 / 3892")).toBeInTheDocument()
    expect(screen.getByText("当前版本：0.1.0")).toBeInTheDocument()
    // Uptime renders as HH:MM:SS for a start ~65s ago.
    expect(screen.getByText(/^00:01:0\d$/)).toBeInTheDocument()
  })

  it("reports an available update", async () => {
    const user = userEvent.setup()
    vi.mocked(checkForUpdate).mockResolvedValue({
      state: "available",
      currentVersion: "0.1.0",
      latestVersion: "v0.2.0",
      releaseUrl: "https://example.com/release",
      publishedAt: "2026-09-01T10:00:00Z",
      message: "",
    })
    renderAbout()

    await user.click(await screen.findByRole("button", { name: "检查更新" }))

    await waitFor(() => {
      expect(checkForUpdate).toHaveBeenCalledTimes(1)
    })
    expect(await screen.findByText("有新版本可用。")).toBeInTheDocument()
    expect(screen.getByText("最新版本：v0.2.0")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "打开发布页面" })).toHaveAttribute(
      "href",
      "https://example.com/release",
    )
  })

  it("reports an unconfigured update check", async () => {
    const user = userEvent.setup()
    vi.mocked(checkForUpdate).mockResolvedValue({
      state: "unconfigured",
      currentVersion: "0.1.0",
      latestVersion: "",
      releaseUrl: "",
      publishedAt: "",
      message: "update: repository is not configured",
    })
    renderAbout()

    await user.click(await screen.findByRole("button", { name: "检查更新" }))

    expect(await screen.findByText("尚未配置发布仓库，暂时无法检查更新。")).toBeInTheDocument()
  })

  it("reports an up-to-date version", async () => {
    const user = userEvent.setup()
    vi.mocked(checkForUpdate).mockResolvedValue({
      state: "up-to-date",
      currentVersion: "0.1.0",
      latestVersion: "v0.1.0",
      releaseUrl: "",
      publishedAt: "",
      message: "",
    })
    renderAbout()

    await user.click(await screen.findByRole("button", { name: "检查更新" }))

    expect(await screen.findByText("已是最新版本。")).toBeInTheDocument()
  })
})
