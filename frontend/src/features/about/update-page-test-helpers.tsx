import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render } from "@testing-library/react"
import { vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { getRuntimeInfo } from "../../services/omniproxy"
import {
  getApplicationUpdate,
  getUpdatePreferences,
  setAutomaticUpdateCheck,
  setAutomaticUpdateDownload,
} from "../../services/updates"
import { UpdateCard } from "./update-card"

export const info = {
  version: "2.0.0-beta.7",
  appName: "OmniProxy",
  description: "Local AI API gateway",
  appId: "com.omniproxy.app.test",
  profile: "test",
  dataDir: "/tmp/omniproxy-test",
  logDir: "/tmp/omniproxy-test/logs",
  proxyPort: 3002,
  controlPort: 3892,
  goVersion: "go1.26.2",
  wailsVersion: "v3.0.0-beta.30",
  platform: "windows/amd64",
  sqliteVersion: "3.50.4",
  startedAt: "2026-10-19T00:00:00Z",
}
export const preferences = {
  channel: "beta",
  repo: "mibgb65-cloud/OmniProxy",
  autoCheck: true,
  autoDownload: false,
}

export function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <UpdateCard />
      </QueryClientProvider>
    </I18nProvider>,
  )
}

export function resetUpdatePage() {
  vi.resetAllMocks()
  localStorage.clear()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getRuntimeInfo).mockResolvedValue(info)
  vi.mocked(getUpdatePreferences).mockResolvedValue(preferences)
  vi.mocked(setAutomaticUpdateCheck).mockResolvedValue({ ...preferences, autoCheck: false })
  vi.mocked(setAutomaticUpdateDownload).mockResolvedValue({ ...preferences, autoDownload: true })
  vi.mocked(getApplicationUpdate).mockResolvedValue({
    currentVersion: "2.0.0-beta.6",
    checkState: "idle",
    checkError: "",
    release: null,
    download: { state: "idle", version: "", received: 0, total: 0, percent: 0, error: "" },
    canInstall: false,
  })
}
