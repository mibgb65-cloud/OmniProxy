import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterContextProvider,
} from "@tanstack/react-router"
import { render } from "@testing-library/react"
import type { ReactNode } from "react"
import { vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import {
  readDashboard,
  type DashboardInfo,
  type DashboardQuery,
  type DashboardSummary,
} from "../../services/dashboard"
import { listAccounts, listProviders } from "../../services/omniproxy"
import { listRoutes } from "../../services/routing"
import { listAgentClients } from "../../services/agents"
import { readSubscriptionQuota } from "../../services/subscription"
import { readAccountBalance } from "../../services/balance"
import { accounts, providers, quota, balance } from "../quota/quota-test-helpers"
import { DashboardView } from "./dashboard-view"

export { accounts, providers, quota, balance }
export const summary: DashboardSummary = {
  requests: 12,
  success: 8,
  failed: 2,
  cancelled: 1,
  pending: 1,
  inputTokens: "900719925474099312345",
  outputTokens: "2000",
  totalTokens: "900719925474099314345",
  completeUsage: 9,
  successRate: 80,
  averageLatencyMs: 1500,
}
export function snapshot(query: DashboardQuery): DashboardInfo {
  const bounds = query.boundaries ?? []
  return {
    summary,
    points: bounds.slice(0, -1).map((start, index) => ({
      start,
      end: bounds[index + 1]!,
      summary:
        index === bounds.length - 2
          ? summary
          : {
              ...summary,
              requests: 0,
              success: 0,
              failed: 0,
              cancelled: 0,
              pending: 0,
              inputTokens: "0",
              outputTokens: "0",
              totalTokens: "0",
              completeUsage: 0,
              successRate: null,
              averageLatencyMs: null,
            },
    })),
    recent: [
      {
        id: "a".repeat(32),
        startedAt: new Date().toISOString(),
        finishedAt: new Date().toISOString(),
        protocol: "responses",
        transport: "http",
        inputModel: "alias",
        model: "gpt-test",
        stream: false,
        providerId: "openai",
        providerName: "OpenAI",
        accountId: "sub-good",
        accountName: "Member",
        email: "",
        workspaceId: "",
        accountDeleted: false,
        state: "success",
        statusCode: 200,
        errorCode: "",
        latencyMs: 1500,
        inputTokens: "1000",
        outputTokens: "20",
        inputKnown: true,
        outputKnown: true,
        usageComplete: true,
      },
    ],
    historyIncomplete: false,
    capturedFrom: bounds[0] ?? 0,
    updatedAt: new Date().toISOString(),
  }
}

export function resetDashboardServices() {
  vi.mocked(readDashboard).mockImplementation(async (query) => snapshot(query))
  vi.mocked(listAccounts).mockResolvedValue(accounts)
  vi.mocked(listProviders).mockResolvedValue(providers)
  vi.mocked(listRoutes).mockResolvedValue([
    {
      rule: {
        id: "route-one",
        alias: "coding",
        providerId: "openai",
        accountId: "",
        model: "gpt-test",
        enabled: true,
        createdAt: "",
        updatedAt: "",
      },
      providerName: "OpenAI",
      accountName: "",
      errorCode: "provider_disabled",
    },
  ])
  vi.mocked(listAgentClients).mockResolvedValue([
    {
      client: "codex",
      path: "",
      exists: true,
      configured: true,
      provider: "openai",
      canRestore: false,
      revision: "",
      errorCode: "",
    },
  ])
  vi.mocked(readSubscriptionQuota).mockResolvedValue(quota)
  vi.mocked(readAccountBalance).mockResolvedValue(balance)
}

export function renderDashboard(seed?: (client: QueryClient) => void) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  seed?.(client)
  const root = createRootRoute()
  const routes = [
    "/",
    "/requests",
    "/accounts",
    "/accounts/$providerId",
    "/quota",
    "/routing",
    "/gateway",
  ].map((path) => createRoute({ getParentRoute: () => root, path }))
  const router = createRouter({
    routeTree: root.addChildren(routes),
    history: createMemoryHistory({ initialEntries: ["/"] }),
  })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <I18nProvider>
        <QueryClientProvider client={client}>
          <RouterContextProvider router={router}>{children}</RouterContextProvider>
        </QueryClientProvider>
      </I18nProvider>
    )
  }
  return { ...render(<DashboardView />, { wrapper: Wrapper }), client }
}
