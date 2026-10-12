import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import { act, renderHook, waitFor } from "@testing-library/react"
import type { ReactNode } from "react"
import { expect, it, vi } from "vitest"
import { readDashboard } from "../../services/dashboard"
import { snapshot } from "./dashboard-test-helpers"
import { useDashboard } from "./use-dashboard"

vi.mock("../../services/dashboard", () => ({
  readDashboard: vi.fn(async (query) => snapshot(query)),
}))
vi.mock("../../services/omniproxy", async (original) => ({
  ...(await original<typeof import("../../services/omniproxy")>()),
  listAccounts: vi.fn(async () => []),
  listProviders: vi.fn(async () => []),
}))
vi.mock("../../services/routing", async (original) => ({
  ...(await original<typeof import("../../services/routing")>()),
  listRoutes: vi.fn(async () => []),
}))
vi.mock("../../services/agents", async (original) => ({
  ...(await original<typeof import("../../services/agents")>()),
  listAgentClients: vi.fn(async () => []),
}))
vi.mock("@wailsio/runtime", async (original) => ({
  ...(await original<typeof import("@wailsio/runtime")>()),
  Events: { On: vi.fn(() => vi.fn()) },
}))

it("defers activity refresh while hidden and updates the active snapshot on return", async () => {
  let visibility: DocumentVisibilityState = "visible"
  const spy = vi.spyOn(document, "visibilityState", "get").mockImplementation(() => visibility)
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
  const { result, unmount } = renderHook(useDashboard, { wrapper: Wrapper })
  try {
    await waitFor(() => expect(result.current.statistics.isSuccess).toBe(true))
    const callback = vi
      .mocked(Events.On)
      .mock.calls.find(([name]) => name === "activity:updated")?.[1]
    visibility = "hidden"
    await act(async () => {
      callback?.({ name: "activity:updated", data: "", sender: "" })
    })
    expect(readDashboard).toHaveBeenCalledTimes(1)
    visibility = "visible"
    await act(async () => {
      document.dispatchEvent(new Event("visibilitychange"))
    })
    await waitFor(() => expect(readDashboard).toHaveBeenCalledTimes(2))
  } finally {
    unmount()
    spy.mockRestore()
  }
})
