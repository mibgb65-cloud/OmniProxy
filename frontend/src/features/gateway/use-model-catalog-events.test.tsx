import type { PropsWithChildren } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, render, renderHook, screen } from "@testing-library/react"
import { Events } from "@wailsio/runtime"
import { expect, it, vi } from "vitest"
import { useModelCatalogEvents } from "./use-model-catalog-events"
import { AppProviders } from "../../app/providers"

vi.mock("@wailsio/runtime", () => ({ Events: { On: vi.fn() } }))

it("刷新实际目录并清理订阅，不反向触发 Google 发现查询", () => {
  const unsubscribe = vi.fn()
  vi.mocked(Events.On).mockReturnValue(unsubscribe)
  const client = new QueryClient()
  const invalidate = vi.spyOn(client, "invalidateQueries")
  function Wrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
  const { unmount } = renderHook(useModelCatalogEvents, { wrapper: Wrapper })
  const callback = vi.mocked(Events.On).mock.calls.at(-1)?.[1]
  expect(Events.On).toHaveBeenLastCalledWith("modelcatalog:updated", expect.any(Function))
  act(() => callback?.({ name: "modelcatalog:updated", data: { changed: true } }))
  expect(invalidate.mock.calls.map(([options]) => options?.queryKey)).toEqual([
    ["model-catalog"],
    ["default-routes"],
    ["routes"],
    ["agent-clients"],
    ["agent-configured-models"],
  ])
  unmount()
  expect(unsubscribe).toHaveBeenCalledOnce()
  client.clear()
})

it("应用级目录事件在 QueryClientProvider 内订阅，不给纯布局增加查询依赖", () => {
  const unsubscribe = vi.fn()
  vi.mocked(Events.On).mockReturnValue(unsubscribe)
  const { unmount } = render(
    <AppProviders>
      <span>app-ready</span>
    </AppProviders>,
  )
  expect(screen.getByText("app-ready")).toBeInTheDocument()
  expect(Events.On).toHaveBeenLastCalledWith("modelcatalog:updated", expect.any(Function))
  unmount()
  expect(unsubscribe).toHaveBeenCalledOnce()
})
