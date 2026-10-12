import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from "@tanstack/react-router"
import { render } from "@testing-library/react"
import type { ReactNode } from "react"
import { I18nProvider } from "../../i18n/provider"
import type { DefaultModelRoute, RouteItem } from "../../services/routing"
import type { ProviderConfiguration, AccountItem } from "../../services/omniproxy"
import type { listModelCatalog, syncAccountModels } from "../../services/models"
import { Route as RoutingRoute } from "../../app/routes/routing"
import type { RoutingTab } from "./routing-tabs"
import { vi } from "vitest"
const modelMocks = vi.hoisted(() => ({
  listModelCatalog: vi.fn<typeof listModelCatalog>().mockResolvedValue([]),
  syncAccountModels: vi.fn<typeof syncAccountModels>(),
}))
vi.mock("../../services/models", () => modelMocks)
export const listModelCatalogMock = modelMocks.listModelCatalog
export const syncAccountModelsMock = modelMocks.syncAccountModels

export const providers: ProviderConfiguration[] = [
  {
    id: "builtin_deepseek",
    kind: "deepseek",
    source: "builtin",
    name: "DeepSeek",
    baseUrl: "",
    enabled: true,
    createdAt: "",
    updatedAt: "",
  },
  {
    id: "builtin_openai",
    kind: "openai",
    source: "builtin",
    name: "OpenAI",
    baseUrl: "",
    enabled: true,
    createdAt: "",
    updatedAt: "",
  },
]
export const accounts: AccountItem[] = [
  {
    id: "acct_one",
    providerId: "builtin_deepseek",
    name: "My account",
    enabled: true,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "",
  },
]
export const ruleItem: RouteItem = {
  rule: {
    id: "route_one",
    alias: "coding-fast",
    providerId: "builtin_deepseek",
    model: "deepseek-flash",
    accountId: "",
    enabled: true,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "",
  },
  providerName: "DeepSeek",
  accountName: "",
  errorCode: "",
}

export const defaultMapping: DefaultModelRoute = {
  model: "deepseek-flash",
  name: "deepseek-flash",
  providers: [
    { id: "builtin_deepseek", name: "DeepSeek", candidateCount: 1 },
    { id: "backup_ds", name: "备用 DeepSeek", candidateCount: 1 },
  ],
  preferredProviderId: "",
  preferredProviderName: "",
  preview: {
    inputModel: "deepseek-flash",
    matched: false,
    ruleId: "",
    model: "deepseek-flash",
    providerId: "builtin_deepseek",
    providerName: "DeepSeek",
    accountId: "acct_one",
    accountName: "My account",
    candidateCount: 2,
    routable: true,
    errorCode: "",
  },
}

export function renderRouting(tab: RoutingTab = "defaults", entry = `/routing?tab=${tab}`) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const root = createRootRoute()
  const routing = createRoute({
    getParentRoute: () => root,
    path: "/routing",
    component: RoutingRoute.options.component,
    validateSearch: RoutingRoute.options.validateSearch,
  })
  const router = createRouter({
    routeTree: root.addChildren([routing]),
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
