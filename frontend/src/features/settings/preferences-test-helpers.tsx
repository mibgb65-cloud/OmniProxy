import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
} from "@tanstack/react-router"
import { render } from "@testing-library/react"
import { vi } from "vitest"
import { AppSidebar } from "../../components/common/app-sidebar"
import { I18nProvider } from "../../i18n/provider"
import { readRetention, setRetention } from "../../services/activity"
import { getDesktopSettings, getRuntimeInfo, getStorageInfo } from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import { useAppPreferences } from "./app-preferences-state"
import { OpenPreferencesRoute } from "./open-preferences-route"

vi.mock("../../features/gateway/sidebar-gateway-status", () => ({
  SidebarGatewayStatus: () => <section aria-label="网关" />,
}))
vi.mock("../../services/activity", () => ({ readRetention: vi.fn(), setRetention: vi.fn() }))
vi.mock("../../services/omniproxy", () => ({
  getStorageInfo: vi.fn(),
  setDataDir: vi.fn(),
  selectDirectory: vi.fn(),
  cleanStorage: vi.fn(),
  getDesktopSettings: vi.fn(),
  setAutoStart: vi.fn(),
  setCloseBehavior: vi.fn(),
  getRuntimeInfo: vi.fn(),
  checkForUpdate: vi.fn(),
}))
vi.mock("@wailsio/runtime", () => ({ Clipboard: { SetText: vi.fn() } }))
vi.mock("../../services/updates", () => ({
  getUpdatePreferences: vi
    .fn()
    .mockResolvedValue({ channel: "stable", repo: "mibgb65-cloud/OmniProxy" }),
  setUpdateChannel: vi.fn(),
  setAutomaticUpdateCheck: vi.fn(),
  getApplicationUpdate: vi.fn().mockResolvedValue({
    currentVersion: "2.0.0-beta.1",
    checkState: "idle",
    checkError: "",
    release: null,
    download: { state: "idle", version: "", received: 0, total: 0, percent: 0, error: "" },
    canInstall: false,
  }),
  startApplicationUpdate: vi.fn(),
  cancelApplicationUpdate: vi.fn(),
  installApplicationUpdate: vi.fn(),
}))
vi.mock("../../services/legacy-import", () => ({
  previewLegacyImport: vi.fn(),
  importLegacyData: vi.fn(),
}))

export function resetPreferences() {
  vi.clearAllMocks()
  localStorage.clear()
  useUIStore.setState({ language: "zh", sidebarCollapsed: false, fontScale: 100 })
  useAppPreferences.setState({ open: false, tab: "settings" })
  vi.mocked(getStorageInfo).mockResolvedValue({
    dataDir: "C:/test/data",
    logDir: "C:/test/data/logs",
    databasePath: "C:/test/data/app.db",
    pendingDataDir: "",
    items: [],
    totalBytes: 0,
  })
  vi.mocked(getDesktopSettings).mockResolvedValue({
    autoStart: false,
    autoStartAvailable: false,
    closeBehavior: "exit",
  })
  vi.mocked(readRetention).mockResolvedValue({
    maxRecords: 10000,
    storedRecords: 4,
    logCapacity: 5000,
  })
  vi.mocked(setRetention).mockResolvedValue({
    maxRecords: 20000,
    storedRecords: 4,
    logCapacity: 5000,
  })
  vi.mocked(getRuntimeInfo).mockResolvedValue({
    version: "0.1.0",
    appName: "OmniProxy",
    description: "Test",
    appId: "test.preferences",
    profile: "test",
    dataDir: "C:/test/data",
    logDir: "C:/test/data/logs",
    proxyPort: 3002,
    controlPort: 3892,
    goVersion: "go1.26.2",
    wailsVersion: "v3-test",
    sqliteVersion: "3-test",
    platform: "windows/amd64",
    startedAt: new Date().toISOString(),
  })
}

export function renderPreferences(entry = "/gateway?tab=clients") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const root = createRootRoute({
    component: () => (
      <>
        <AppSidebar />
        <main>
          <Outlet />
        </main>
      </>
    ),
  })
  const home = createRoute({
    getParentRoute: () => root,
    path: "/",
    component: () => <p>首页内容</p>,
  })
  const gateway = createRoute({
    getParentRoute: () => root,
    path: "/gateway",
    component: () => <input aria-label="工作草稿" defaultValue="原工作" />,
  })
  const settings = createRoute({
    getParentRoute: () => root,
    path: "/settings",
    component: () => <OpenPreferencesRoute tab="settings" />,
  })
  const about = createRoute({
    getParentRoute: () => root,
    path: "/about",
    component: () => <OpenPreferencesRoute tab="about" />,
  })
  const others = [
    "/accounts",
    "/providers",
    "/routing",
    "/quota",
    "/usage",
    "/requests",
    "/logs",
  ].map((path) => createRoute({ getParentRoute: () => root, path }))
  const router = createRouter({
    routeTree: root.addChildren([home, gateway, settings, about, ...others]),
    history: createMemoryHistory({ initialEntries: [entry] }),
  })
  return {
    ...render(
      <I18nProvider>
        <QueryClientProvider client={client}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </I18nProvider>,
    ),
    router,
    client,
  }
}
