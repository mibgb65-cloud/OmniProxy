import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, cleanup, renderHook } from "@testing-library/react"
import type { ReactNode } from "react"
import { afterEach, beforeEach, expect, it, vi } from "vitest"
import { getGatewayInfo, startGateway, stopGateway, type GatewayInfo } from "../../services/gateway"
import { getAppInfo, listAccounts, listProviders } from "../../services/omniproxy"
import { useGatewayPreparation } from "./use-gateway-preparation"
import { useGatewayRuntime } from "./use-gateway-runtime"

vi.mock("../../services/gateway", () => ({
  getGatewayInfo: vi.fn(),
  startGateway: vi.fn(),
  stopGateway: vi.fn(),
}))
vi.mock("../../services/omniproxy", () => ({
  getAppInfo: vi.fn(),
  listAccounts: vi.fn(),
  listProviders: vi.fn(),
}))
vi.mock("@wailsio/runtime", () => ({ Events: { On: vi.fn(() => vi.fn()) } }))

const stopped: GatewayInfo = {
  runtime: { state: "stopped", address: "", port: 43210, startedAt: "", errorCode: "" },
  models: [],
  protocol: "Chat Completions",
  requestTimeoutSeconds: 120,
}
const running: GatewayInfo = { ...stopped, runtime: { ...stopped.runtime, state: "running" } }
const appInfo = {
  version: "test",
  profile: "test",
  appId: "test",
  dataDir: "",
  logDir: "",
  proxyPort: 43210,
  controlPort: 43211,
}
const advance = async (ms: number) =>
  act(async () => {
    await vi.advanceTimersByTimeAsync(ms)
  })

beforeEach(() => {
  vi.useFakeTimers()
  vi.clearAllMocks()
  vi.mocked(getGatewayInfo).mockResolvedValue(stopped)
  vi.mocked(startGateway).mockResolvedValue(running)
  vi.mocked(stopGateway).mockResolvedValue(stopped)
  vi.mocked(getAppInfo).mockResolvedValue(appInfo)
  vi.mocked(listAccounts).mockResolvedValue([])
  vi.mocked(listProviders).mockResolvedValue([])
})
afterEach(() => {
  cleanup()
  vi.useRealTimers()
})

async function mount() {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: Infinity },
      mutations: { gcTime: Infinity },
    },
  })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  const hook = renderHook(() => ({ page: useGatewayPreparation(), sidebar: useGatewayRuntime() }), {
    wrapper,
  })
  await advance(1)
  return { ...hook, client }
}

it("快速启停立即执行和更新缓存，共享反馈保持650ms并阻止跨入口重复操作", async () => {
  const { result } = await mount()
  act(() => result.current.page.gateway.toggle())
  await advance(0)
  expect(startGateway).toHaveBeenCalledTimes(1)
  expect(result.current.page.gateway.info.data?.runtime.state).toBe("running")
  expect(result.current.page.gateway.statusKey).toBe("gateway.status.starting")
  expect(result.current.sidebar.controlsBusy).toBe(true)
  expect(result.current.page.isRefreshing).toBe(false)
  act(() => result.current.sidebar.toggle())
  expect(stopGateway).not.toHaveBeenCalled()
  await advance(649)
  expect(result.current.page.gateway.controlsBusy).toBe(true)
  await advance(2)
  expect(result.current.page.gateway.controlsBusy).toBe(false)
  expect(result.current.sidebar.controlsBusy).toBe(false)
  act(() => result.current.sidebar.toggle())
  await advance(0)
  expect(stopGateway).toHaveBeenCalledTimes(1)
  expect(result.current.page.gateway.info.data?.runtime.state).toBe("stopped")
  expect(result.current.page.gateway.statusKey).toBe("gateway.status.stopping")
  await advance(651)
  expect(result.current.page.gateway.statusKey).toBe("gateway.status.stopped")
})

it("慢启停不会在实际完成后再叠加一轮反馈等待", async () => {
  let finish!: (value: GatewayInfo) => void
  vi.mocked(startGateway).mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        finish = resolve
      }),
  )
  const { result } = await mount()
  act(() => result.current.page.gateway.toggle())
  await advance(900)
  expect(result.current.page.gateway.controlsBusy).toBe(true)
  act(() => finish(running))
  await advance(1)
  expect(result.current.page.gateway.controlsBusy).toBe(false)
  expect(result.current.sidebar.statusKey).toBe("gateway.status.running")
})

it("失败立即结束加载并提供错误，不等待成功动画时长", async () => {
  vi.mocked(startGateway).mockRejectedValueOnce(new Error("gateway: port_unavailable"))
  const { result } = await mount()
  act(() => result.current.page.gateway.toggle())
  await advance(1)
  expect(result.current.page.gateway.controlsBusy).toBe(false)
  expect(result.current.page.gateway.operationError).toBeInstanceOf(Error)
  expect(result.current.sidebar.controlsBusy).toBe(false)
})

it("网关后台状态读取不会触发刷新动画，手动刷新才显示反馈", async () => {
  const { result, client } = await mount()
  let finish!: (value: GatewayInfo) => void
  vi.mocked(getGatewayInfo).mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        finish = resolve
      }),
  )
  act(() => {
    void client.invalidateQueries({ queryKey: ["gateway-runtime"] })
  })
  await advance(1)
  expect(client.isFetching({ queryKey: ["gateway-runtime"] })).toBe(1)
  expect(result.current.page.isRefreshing).toBe(false)
  act(() => finish(stopped))
  await advance(1)
  act(() => result.current.page.refresh())
  await advance(1)
  expect(result.current.page.isRefreshing).toBe(true)
  await advance(651)
  expect(result.current.page.isRefreshing).toBe(false)
})
