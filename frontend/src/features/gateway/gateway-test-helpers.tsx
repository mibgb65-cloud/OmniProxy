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
import type { ReactNode } from "react"
import { vi } from "vitest"
import { getGatewayInfo, startGateway, stopGateway, type GatewayInfo } from "../../services/gateway"
import { I18nProvider } from "../../i18n/provider"
import type { AccountItem, AppInfo, ProviderConfiguration } from "../../services/omniproxy"
import { Route as GatewayRoute } from "../../app/routes/gateway"
import { SidebarGatewayStatus } from "./sidebar-gateway-status"
import type { GatewayTab } from "./gateway-tabs"
const listAgentClientsMock = vi.hoisted(() => vi.fn().mockResolvedValue([]))
vi.mock("../../services/failover", () => ({
  getFailoverSettings: vi.fn().mockResolvedValue({ enabled: true, maxAttempts: 3 }),
  setFailoverEnabled: vi.fn(async (enabled: boolean) => ({ enabled, maxAttempts: 3 })),
  getFailoverHealth: vi.fn().mockResolvedValue([]),
}))
const readDashboardMock = vi.hoisted(() => vi.fn())
const getConfiguredAgentModelsMock = vi.hoisted(() =>
  vi.fn().mockResolvedValue({ models: [], defaultModel: "" }),
)
const previewAgentConfigMock = vi.hoisted(() => vi.fn())
const previewAgentPlanMock = vi.hoisted(() => vi.fn())
const applyAgentConfigMock = vi.hoisted(() => vi.fn())
const copyAgentKeyMock = vi.hoisted(() => vi.fn())
const restoreAgentConfigMock = vi.hoisted(() => vi.fn())
const setAgentEnabledMock = vi.hoisted(() => vi.fn())
const rereadAgentConfigMock = vi.hoisted(() => vi.fn())
const getAgentInstallationsMock = vi.hoisted(() => vi.fn())
const refreshAgentInstallationsMock = vi.hoisted(() => vi.fn())
const installAgentMock = vi.hoisted(() => vi.fn())
const openAgentInstallationOfficialMock = vi.hoisted(() => vi.fn())
const openClaudeCodeMock = vi.hoisted(() => vi.fn())
const openAgentClientMock = vi.hoisted(() => vi.fn())
const checkAgentUpdateMock = vi.hoisted(() => vi.fn())
const updateAgentMock = vi.hoisted(() => vi.fn())
const openAgentUpdateMock = vi.hoisted(() => vi.fn())
const agentErrorMock = vi.hoisted(() => vi.fn(() => "failed"))
export {
  readDashboardMock,
  listAgentClientsMock,
  getConfiguredAgentModelsMock,
  previewAgentConfigMock,
  previewAgentPlanMock,
  applyAgentConfigMock,
  copyAgentKeyMock,
  restoreAgentConfigMock,
  setAgentEnabledMock,
  rereadAgentConfigMock,
  getAgentInstallationsMock,
  refreshAgentInstallationsMock,
  installAgentMock,
  openAgentInstallationOfficialMock,
  openClaudeCodeMock,
  openAgentClientMock,
  checkAgentUpdateMock,
  updateAgentMock,
  openAgentUpdateMock,
  agentErrorMock,
}
vi.mock("../../services/dashboard", () => ({ readDashboard: readDashboardMock }))
vi.mock("../../services/agent-installation", async () => ({
  getAgentInstallations: getAgentInstallationsMock,
  refreshAgentInstallations: refreshAgentInstallationsMock,
  installAgent: installAgentMock,
  openAgentInstallationOfficial: openAgentInstallationOfficialMock,
  openClaudeCode: openClaudeCodeMock,
  openAgentClient: openAgentClientMock,
  checkAgentUpdate: checkAgentUpdateMock,
  updateAgent: updateAgentMock,
  openAgentUpdate: openAgentUpdateMock,
  agentInstallationError: (await import("../../services/agent-installation-error"))
    .agentInstallationError,
}))
vi.mock("../../services/agents", () => ({
  listAgentClients: listAgentClientsMock,
  getConfiguredAgentModels: getConfiguredAgentModelsMock,
  previewAgentConfig: previewAgentConfigMock,
  previewAgentPlan: previewAgentPlanMock,
  applyAgentConfig: applyAgentConfigMock,
  restoreAgentConfig: restoreAgentConfigMock,
  setAgentEnabled: setAgentEnabledMock,
  agentError: agentErrorMock,
  agentTitle: () => "agents.codex",
  copyAgentKey: copyAgentKeyMock,
  rereadAgentConfig: rereadAgentConfigMock,
}))

export const appInfo: AppInfo = {
  version: "0.1.0",
  profile: "development",
  appId: "test.gateway",
  dataDir: "",
  logDir: "",
  proxyPort: 43210,
  controlPort: 3891,
}
export const providers: ProviderConfiguration[] = [
  { id: "enabled", enabled: true },
  { id: "disabled", enabled: false },
].map((item) => ({
  ...item,
  kind: "deepseek",
  source: "builtin",
  name: item.id,
  baseUrl: "",
  createdAt: "",
  updatedAt: "",
}))
export const accounts: AccountItem[] = [
  { id: "one", providerId: "enabled", enabled: true },
  { id: "two", providerId: "disabled", enabled: true },
  { id: "three", providerId: "enabled", enabled: false },
].map((item) => ({ ...item, name: item.id, createdAt: "", updatedAt: "" }))

export function renderGateway(
  tab: GatewayTab = "overview",
  {
    entry = `/gateway?tab=${tab}`,
    sidebar = false,
    scrollViewport = false,
  }: {
    entry?: string
    sidebar?: boolean
    scrollViewport?: boolean
  } = {},
) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const rootRoute = createRootRoute({
    component: () => (
      <>
        {sidebar ? <SidebarGatewayStatus collapsed={false} /> : null}
        {scrollViewport ? (
          <main className="scroll-region__viewport">
            <Outlet />
          </main>
        ) : (
          <Outlet />
        )}
      </>
    ),
  })
  const gatewayRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: "/gateway",
    component: GatewayRoute.options.component,
    validateSearch: GatewayRoute.options.validateSearch,
  })
  const routes = ["/providers", "/accounts", "/routing", "/requests", "/logs"].map((path) =>
    createRoute({ getParentRoute: () => rootRoute, path }),
  )
  const router = createRouter({
    routeTree: rootRoute.addChildren([gatewayRoute, ...routes]),
    history: createMemoryHistory({ initialEntries: [entry] }),
  })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <I18nProvider>
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      </I18nProvider>
    )
  }
  return { ...render(<RouterProvider router={router} />, { wrapper: Wrapper }), router, client }
}

export const gatewayInfo: GatewayInfo = {
  runtime: { state: "stopped", address: "", port: appInfo.proxyPort, startedAt: "", errorCode: "" },
  models: ["deepseek-flash", "deepseek-v4-pro"],
  protocol: "Chat Completions",
  requestTimeoutSeconds: 120,
}

export function resetGatewayRuntime() {
  readDashboardMock.mockReturnValue(new Promise(() => {}))
  const installations = {
    items: ["codex-cli", "codex-desktop", "claude-code", "claude-desktop"].map((id) => ({
      id,
      detection: { state: "installed", version: "1.2.3", path: "", errorCode: "" },
      installState: "idle",
      errorCode: "",
      officialURL: "https://example.invalid",
    })),
    busy: false,
    installerAvailable: true,
  }
  getAgentInstallationsMock.mockResolvedValue(installations)
  refreshAgentInstallationsMock.mockResolvedValue(installations)
  vi.mocked(getGatewayInfo).mockResolvedValue(gatewayInfo)
  vi.mocked(startGateway).mockResolvedValue({
    ...gatewayInfo,
    runtime: { ...gatewayInfo.runtime, state: "running", address: "http://127.0.0.1:43210" },
  })
  vi.mocked(stopGateway).mockResolvedValue(gatewayInfo)
}
