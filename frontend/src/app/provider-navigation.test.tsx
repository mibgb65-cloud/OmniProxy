import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from "@tanstack/react-router"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../i18n/provider"
import { getProviderCatalog, listProviders } from "../services/omniproxy"
import { useUIStore } from "../stores/ui-store"
import { Route as ProvidersRoute } from "./routes/providers"

vi.mock("../services/omniproxy", () => ({
  getProviderCatalog: vi.fn(),
  listProviders: vi.fn(),
  createProvider: vi.fn(),
  updateProvider: vi.fn(),
  setProviderEnabled: vi.fn(),
  probeProvider: vi.fn(),
  deleteProvider: vi.fn(),
}))

const builtin = {
  id: "builtin_openai",
  kind: "openai",
  source: "builtin",
  name: "openai",
  baseUrl: "",
  enabled: true,
  proxyMode: "inherit",
  createdAt: "2026-09-28T00:00:00Z",
  updatedAt: "2026-09-28T00:00:00Z",
}

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getProviderCatalog).mockResolvedValue([
    {
      kind: "openai",
      name: "openai",
      defaultBaseUrl: "https://api.example.test/v1",
      models: ["GPT-5.4"],
      authMethods: ["api_key"],
      plannedAuthMethods: ["subscription_login"],
    },
  ])
  vi.mocked(listProviders).mockResolvedValue([builtin])
})

const root = createRootRoute()
const providersRoute = createRoute({
  getParentRoute: () => root,
  path: "/providers",
  component: ProvidersRoute.options.component,
})
const accountsRoute = createRoute({
  getParentRoute: () => root,
  path: "/accounts/$providerId",
  component: AccountsStub,
})

function AccountsStub() {
  const { providerId } = accountsRoute.useParams()
  return <p>账号页面 {providerId}</p>
}

it("提供商卡片跳转到该提供商的账号页面", async () => {
  const user = userEvent.setup()
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const router = createRouter({
    routeTree: root.addChildren([providersRoute, accountsRoute]),
    history: createMemoryHistory({ initialEntries: ["/providers"] }),
  })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </I18nProvider>,
  )
  const title = await screen.findByRole("button", { name: "查看OpenAI的账号" })
  await user.click(title.closest("[data-slot='card']")!)
  expect(await screen.findByText("账号页面 builtin_openai")).toBeInTheDocument()
  expect(router.state.location.pathname).toBe("/accounts/builtin_openai")
})
