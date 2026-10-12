import { act, fireEvent, render, screen, waitFor } from "@testing-library/react"
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from "@tanstack/react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { Route as RootRoute } from "./routes/__root"

vi.mock("../components/common/app-sidebar", () => ({ AppSidebar: () => <nav>Sidebar</nav> }))
vi.mock("../components/common/app-titlebar", () => ({
  AppTitlebar: () => <header>Titlebar</header>,
}))

function renderPages(settingsLoader?: () => Promise<void>) {
  const root = createRootRoute({ component: RootRoute.options.component })
  const routes = ["/settings", "/accounts/$providerId"].map((path) =>
    createRoute({
      getParentRoute: () => root,
      path,
      component: () => <div>Page</div>,
      validateSearch: (search: Record<string, unknown>) => search,
      loader: path === "/settings" ? settingsLoader : undefined,
    }),
  )
  const router = createRouter({
    routeTree: root.addChildren(routes),
    history: createMemoryHistory({ initialEntries: ["/accounts/openai"] }),
  })
  render(<RouterProvider router={router} />)
  return router
}

describe("根布局页面滚动隔离", () => {
  beforeEach(() => vi.spyOn(window, "scrollTo").mockImplementation(() => {}))
  afterEach(() => vi.restoreAllMocks())
  it("动态账号详情、普通页面和前进后退都恢复各自的位置", async () => {
    const router = renderPages()
    const viewport = await screen.findByRole("main")
    Object.defineProperties(viewport, {
      clientHeight: { configurable: true, value: 100 },
      scrollHeight: { configurable: true, value: 1000 },
    })
    const scrollTo = (top: number) => {
      viewport.scrollTop = top
      fireEvent.scroll(viewport)
    }
    scrollTo(360)
    await act(async () =>
      router.navigate({ to: "/accounts/$providerId", params: { providerId: "deepseek" } }),
    )
    expect(screen.getByRole("main")).toBe(viewport)
    expect(viewport.scrollTop).toBe(0)
    scrollTo(180)
    await act(async () => router.navigate({ to: "/settings" }))
    expect(viewport.scrollTop).toBe(0)
    scrollTo(90)
    await act(async () => router.history.back())
    await waitFor(() => expect(viewport.scrollTop).toBe(180))
    await act(async () => router.history.back())
    await waitFor(() => expect(viewport.scrollTop).toBe(360))
    await act(async () => router.history.forward())
    await waitFor(() => expect(viewport.scrollTop).toBe(180))
  })

  it("同页查询参数变化不重置滚动，也不污染其他页面记录", async () => {
    const router = renderPages()
    const viewport = await screen.findByRole("main")
    Object.defineProperties(viewport, {
      clientHeight: { configurable: true, value: 100 },
      scrollHeight: { configurable: true, value: 1000 },
    })
    viewport.scrollTop = 240
    fireEvent.scroll(viewport)
    await act(async () =>
      router.navigate({
        to: "/accounts/$providerId",
        params: { providerId: "openai" },
        search: { view: "list" },
      }),
    )
    expect(viewport.scrollTop).toBe(240)
    await act(async () => router.navigate({ to: "/settings" }))
    expect(viewport.scrollTop).toBe(0)
    await act(async () =>
      router.navigate({ to: "/accounts/$providerId", params: { providerId: "openai" } }),
    )
    expect(viewport.scrollTop).toBe(240)
  })

  it("导航等待期间保留当前页位置，目标页面提交后才恢复", async () => {
    let finish!: () => void
    const router = renderPages(
      () =>
        new Promise<void>((resolve) => {
          finish = resolve
        }),
    )
    const viewport = await screen.findByRole("main")
    Object.defineProperties(viewport, {
      clientHeight: { configurable: true, value: 100 },
      scrollHeight: { configurable: true, value: 1000 },
    })
    viewport.scrollTop = 420
    fireEvent.scroll(viewport)
    let navigation!: Promise<void>
    act(() => {
      navigation = router.navigate({ to: "/settings" })
    })
    await waitFor(() => expect(finish).toBeTypeOf("function"))
    expect(viewport.scrollTop).toBe(420)
    await act(async () => {
      finish()
      await navigation
    })
    expect(viewport.scrollTop).toBe(0)
    await act(async () => router.history.back())
    await waitFor(() => expect(viewport.scrollTop).toBe(420))
  })
})
