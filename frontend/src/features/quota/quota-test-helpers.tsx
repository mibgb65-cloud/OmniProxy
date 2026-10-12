import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterContextProvider,
} from "@tanstack/react-router"
import { render } from "@testing-library/react"
import type { ReactNode } from "react"
import { vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import {
  listAccounts,
  listProviders,
  type AccountItem,
  type ProviderConfiguration,
} from "../../services/omniproxy"
import { readSubscriptionQuota, type SubscriptionQuota } from "../../services/subscription"
import { readAccountBalance, type AccountBalanceInfo } from "../../services/balance"
import { QuotaView } from "./quota-view"

export const providers: ProviderConfiguration[] = [
  { id: "openai", kind: "openai", name: "OpenAI" },
  { id: "deepseek", kind: "deepseek", name: "DeepSeek" },
  { id: "anthropic", kind: "anthropic", name: "Anthropic" },
].map((item) => ({
  ...item,
  source: "builtin",
  enabled: true,
  baseUrl: "",
  createdAt: "",
  updatedAt: "",
}))

export const accounts: AccountItem[] = [
  {
    id: "sub-good",
    providerId: "openai",
    authMethod: "subscription_login",
    authEmail: "member@example.invalid",
    authAccountId: "workspace-one",
    authState: "authenticated",
    authPlan: "plus",
    enabled: true,
  },
  {
    id: "sub-empty",
    providerId: "openai",
    authMethod: "subscription_login",
    authEmail: "empty@example.invalid",
    authState: "authenticated",
    authPlan: "pro",
    enabled: true,
  },
  {
    id: "sub-out",
    providerId: "openai",
    authMethod: "subscription_login",
    authEmail: "out@example.invalid",
    authState: "logged_out",
    authPlan: "plus",
    enabled: true,
  },
  { id: "balance-good", providerId: "deepseek", authMethod: "api_key", enabled: false },
  { id: "balance-error", providerId: "deepseek", authMethod: "api_key", enabled: true },
  { id: "unsupported", providerId: "anthropic", authMethod: "api_key", enabled: true },
].map((item) => ({
  ...item,
  name: item.id,
  createdAt: "2026-10-09T00:00:00Z",
  updatedAt: "2026-10-09T00:00:00Z",
}))

export const quota: SubscriptionQuota = {
  plan: "plus",
  fiveHourEnabled: true,
  fiveHour: { remainingPercent: 75, resetsAt: "2099-10-10T00:00:00Z" },
  weekly: { remainingPercent: 40, resetsAt: "2099-10-15T00:00:00Z" },
  fetchedAt: "2026-10-09T00:00:00Z",
}
export const balance: AccountBalanceInfo = {
  available: true,
  entries: [
    { currency: "CNY", total: "110.000000000000000001", granted: "10.00", toppedUp: "100.00" },
    { currency: "USD", total: "0.03", granted: "0.00", toppedUp: "0.03" },
  ],
  fetchedAt: "2026-10-09T00:00:00Z",
}

export function resetQuotaServices() {
  vi.mocked(listAccounts).mockResolvedValue(accounts)
  vi.mocked(listProviders).mockResolvedValue(providers)
  vi.mocked(readSubscriptionQuota).mockImplementation(async (id) =>
    id === "sub-empty"
      ? {
          ...quota,
          plan: "pro",
          fiveHourEnabled: false,
          fiveHour: null,
          weekly: { ...quota.weekly!, remainingPercent: 0 },
        }
      : quota,
  )
  vi.mocked(readAccountBalance).mockImplementation(async (id) => {
    if (id === "balance-error") throw new Error("private balance detail")
    return balance
  })
}

export function renderQuota(extra?: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const root = createRootRoute()
  const routes = ["/quota", "/accounts", "/accounts/$providerId"].map((path) =>
    createRoute({ getParentRoute: () => root, path }),
  )
  const router = createRouter({
    routeTree: root.addChildren(routes),
    history: createMemoryHistory({ initialEntries: ["/quota"] }),
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
  return {
    ...render(
      <>
        <QuotaView />
        {extra}
      </>,
      { wrapper: Wrapper },
    ),
    client,
  }
}
