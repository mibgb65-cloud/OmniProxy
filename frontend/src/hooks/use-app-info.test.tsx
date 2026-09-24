import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { renderHook, waitFor } from "@testing-library/react"
import type { ReactNode } from "react"
import { describe, expect, it, vi } from "vitest"
import { useAppInfo } from "./use-app-info"

vi.mock("../services/omniproxy", () => ({
  getAppInfo: vi.fn().mockResolvedValue({
    version: "0.1.0",
    profile: "test",
    appId: "com.omniproxy.app.test",
    dataDir: "/tmp/omniproxy-test",
    logDir: "/tmp/omniproxy-test/logs",
    proxyPort: 3002,
    controlPort: 3892,
  }),
}))

function createWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
}

describe("useAppInfo", () => {
  it("returns the app info from the Go backend", async () => {
    const { result } = renderHook(() => useAppInfo(), { wrapper: createWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data?.version).toBe("0.1.0")
    expect(result.current.data?.profile).toBe("test")
    expect(result.current.data?.controlPort).toBe(3892)
  })
})
