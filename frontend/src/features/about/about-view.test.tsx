import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { checkForUpdate, getRuntimeInfo, type RuntimeInfo } from "../../services/omniproxy"
import { AboutView } from "./about-view"
import { UpdateCard } from "./update-card"
import { getUpdatePreferences, setUpdateChannel } from "../../services/updates"

vi.mock("../../services/updates", () => ({
  getUpdatePreferences: vi.fn(),
  setUpdateChannel: vi.fn(),
}))

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

function renderView(view: ReactNode = <AboutView />) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <I18nProvider>
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      </I18nProvider>
    )
  }
  return render(view, { wrapper: Wrapper })
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getRuntimeInfo).mockResolvedValue(runtimeInfo)
  vi.mocked(getUpdatePreferences).mockResolvedValue({
    channel: "stable",
    repo: "mibgb65-cloud/OmniProxy",
  })
  vi.mocked(setUpdateChannel).mockResolvedValue({
    channel: "beta",
    repo: "mibgb65-cloud/OmniProxy",
  })
})

describe("AboutView", () => {
  it("renders runtime details", async () => {
    renderView()

    await waitFor(() => {
      expect(screen.getByText("go1.26.2")).toBeInTheDocument()
    })
    expect(screen.getByText("v3.0.0-beta.25")).toBeInTheDocument()
    expect(screen.getByText("windows/amd64")).toBeInTheDocument()
    expect(screen.getByText("3.50.4")).toBeInTheDocument()
    expect(screen.getByText("3002 / 3892")).toBeInTheDocument()
    expect(screen.getByText("0.1.0")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "检查更新" })).not.toBeInTheDocument()
    expect(checkForUpdate).not.toHaveBeenCalled()
    // Uptime renders as HH:MM:SS for a start ~65s ago.
    expect(screen.getByText(/^00:01:0\d$/)).toBeInTheDocument()
  })

  it.each([
    ["zh", ["官网", "开源仓库", "免责协议", "使用协议", "隐私协议"], "待配置"],
    [
      "en",
      ["Official website", "Source repository", "Disclaimer", "Terms of use", "Privacy policy"],
      "Coming soon",
    ],
  ] as const)(
    "%s 展示五个待配置资源入口，不生成跳转或协议正文",
    async (language, labels, pending) => {
      useUIStore.setState({ language })
      renderView()
      for (const label of labels) {
        expect(
          screen.getByRole("button", { name: new RegExp(`^${label}\\s*${pending}$`) }),
        ).toBeDisabled()
      }
      expect(screen.queryByRole("link")).not.toBeInTheDocument()
      expect(checkForUpdate).not.toHaveBeenCalled()
      expect(await screen.findByText("go1.26.2")).toBeInTheDocument()
    },
  )
})

describe("UpdateCard", () => {
  it("保存 Beta 渠道后清除旧渠道结果，不自动联网检查", async () => {
    const user = userEvent.setup()
    vi.mocked(checkForUpdate).mockResolvedValue({
      state: "up-to-date",
      currentVersion: "2.0.0-beta.1",
      latestVersion: "v1.3.1",
      releaseUrl: "",
      publishedAt: "",
      message: "",
    })
    renderView(<UpdateCard />)
    await user.click(await screen.findByRole("button", { name: "检查更新" }))
    expect(await screen.findByText("已是最新版本。")).toBeInTheDocument()
    await user.click(screen.getByRole("combobox", { name: "更新渠道" }))
    await user.click(await screen.findByRole("option", { name: "正式版与 Beta 测试版" }))
    await waitFor(() => expect(setUpdateChannel).toHaveBeenCalledWith("beta", expect.anything()))
    await waitFor(() => expect(screen.queryByText("已是最新版本。")).not.toBeInTheDocument())
    expect(checkForUpdate).toHaveBeenCalledTimes(1)
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
    renderView(<UpdateCard />)

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
    renderView(<UpdateCard />)

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
    renderView(<UpdateCard />)

    await user.click(await screen.findByRole("button", { name: "检查更新" }))

    expect(await screen.findByText("已是最新版本。")).toBeInTheDocument()
  })
})
