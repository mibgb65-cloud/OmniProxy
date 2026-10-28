import { createMemoryHistory, createRouter, RouterContextProvider } from "@tanstack/react-router"
import {
  act,
  fireEvent,
  isInaccessible,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { routeTree } from "../../app/routeTree.gen"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { stubNavRowMetrics } from "../../test/nav-row-metrics"
import { AppSidebar } from "./app-sidebar"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

vi.mock("../../services/updates", () => ({
  getApplicationUpdate: vi.fn().mockResolvedValue({
    currentVersion: "2.0.0-beta.1",
    checkState: "idle",
    checkError: "",
    release: null,
    download: { state: "idle", version: "", received: 0, total: 0, percent: 0, error: "" },
    canInstall: false,
  }),
}))

vi.mock("../../features/gateway/sidebar-gateway-status", () => ({
  SidebarGatewayStatus: () => <section aria-label="网关" />,
}))

function renderSidebar() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const router = createRouter({
    routeTree,
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
  return { ...render(<AppSidebar />, { wrapper: Wrapper }), router }
}

describe("AppSidebar", () => {
  beforeEach(() => {
    localStorage.clear()
    useUIStore.setState({ language: "zh", sidebarCollapsed: false })
  })

  it("renders the primary navigation in the selected language", () => {
    renderSidebar()
    expect(screen.getByText("OmniProxy")).toBeInTheDocument()
    expect(screen.getByText("仪表盘")).toBeInTheDocument()
    expect(screen.getByText("账号")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "设置与关于" })).toBeInTheDocument()
    expect(screen.queryByRole("link", { name: "设置" })).not.toBeInTheDocument()
    expect(screen.queryByRole("link", { name: "关于" })).not.toBeInTheDocument()
    expect(screen.queryByRole("group", { name: "语言" })).not.toBeInTheDocument()
  })

  it("switches the whole navigation to English", async () => {
    renderSidebar()

    act(() => useUIStore.getState().setLanguage("en"))

    await waitFor(() => {
      expect(screen.getByText("Dashboard")).toBeInTheDocument()
    })
    expect(screen.getByText("Accounts")).toBeInTheDocument()
    expect(screen.queryByText("仪表盘")).not.toBeInTheDocument()
  })

  it.each([false, true])(
    "blocks native navigation dragging while preserving mouse and keyboard navigation (collapsed=%s)",
    async (sidebarCollapsed) => {
      useUIStore.setState({ sidebarCollapsed })
      const user = userEvent.setup()
      const { router, container } = renderSidebar()
      for (const link of screen.getAllByRole("link")) {
        expect(link).toHaveAttribute("draggable", "false")
        expect(fireEvent.dragStart(link)).toBe(false)
      }
      expect(container.querySelector<HTMLImageElement>("aside img")?.draggable).toBe(false)
      expect(router.history.location.pathname).toBe("/")
      await user.click(screen.getByRole("link", { name: "提供商" }))
      await waitFor(() => expect(router.history.location.pathname).toBe("/providers"))
      act(() => screen.getByRole("link", { name: "账号" }).focus())
      await user.keyboard("{Enter}")
      await waitFor(() => expect(router.history.location.pathname).toBe("/accounts"))
    },
  )

  it("keeps the only brand mark and the collapse toggle in the sidebar header", async () => {
    const user = userEvent.setup()
    renderSidebar()
    const brand = screen.getByTestId("sidebar-brand")

    expect(brand.querySelector("img")).not.toBeNull()
    expect(screen.getAllByText("OmniProxy")).toHaveLength(1)
    expect(within(brand).getByRole("button", { name: "折叠" })).toBeInTheDocument()
    expect(within(brand).queryByRole("button", { name: "展开" })).not.toBeInTheDocument()

    await user.click(within(brand).getByRole("button", { name: "折叠" }))

    // 收起后品牌图标仍是唯一的品牌标识，整行变成展开按钮，文字不再占位。
    expect(screen.getByTestId("sidebar-brand").querySelector("img")).not.toBeNull()
    expect(
      within(screen.getByTestId("sidebar-brand")).getByRole("button", { name: "展开" }),
    ).toBeInTheDocument()
    expect(screen.queryByText("OmniProxy")).not.toBeInTheDocument()
  })

  it("collapses to icons only and restores labels", async () => {
    const user = userEvent.setup()
    renderSidebar()

    await user.click(screen.getByRole("button", { name: "折叠" }))
    expect(isInaccessible(screen.getByText("仪表盘"))).toBe(true)
    expect(screen.getByRole("link", { name: "仪表盘" })).toHaveAttribute("href", "/")
    expect(screen.getAllByRole("link")).toHaveLength(9)
    expect(screen.getByRole("button", { name: "展开" })).toHaveAttribute("aria-expanded", "false")

    await user.click(screen.getByRole("button", { name: "展开" }))
    expect(screen.getByText("仪表盘")).toBeInTheDocument()
    expect(isInaccessible(screen.getByText("仪表盘"))).toBe(false)
    expect(screen.getByRole("button", { name: "折叠" })).toHaveAttribute("aria-expanded", "true")
  })

  it("slides the selected background to the new item instead of swapping it", async () => {
    const user = userEvent.setup()
    const restoreMetrics = stubNavRowMetrics()
    try {
      const { container } = renderSidebar()

      // 「提供商」是第三行：药丸位移到 2 ×（行高 36 + 间距 4），高度等于行高。
      await user.click(screen.getByRole("link", { name: "提供商" }))
      await waitFor(() =>
        expect(
          container.querySelector<HTMLElement>(".sidebar-nav-indicator")?.style.transform,
        ).toBe("translateY(80px)"),
      )
      const background = container.querySelector(".sidebar-nav-indicator")
      expect((background as HTMLElement).style.height).toBe("36px")

      // 回到第一行：仍是同一个背景元素，只是位置变了。
      await user.click(screen.getByRole("link", { name: "仪表盘" }))
      await waitFor(() => expect(background).toHaveStyle({ transform: "translateY(0px)" }))
      expect(container.querySelector(".sidebar-nav-indicator")).toBe(background)
    } finally {
      restoreMetrics()
    }
  })

  it("keeps the selected background in place when the sidebar collapses", async () => {
    const user = userEvent.setup()
    const restoreMetrics = stubNavRowMetrics()
    try {
      const { container } = renderSidebar()
      const background = container.querySelector<HTMLElement>(".sidebar-nav-indicator")
      expect(background?.style.transform).toBe("translateY(0px)")

      // 收展只改变宽度：背景块沿用同一元素与同一行坐标，不被重建或错位。
      await user.click(screen.getByRole("button", { name: "折叠" }))
      expect(background?.style.transform).toBe("translateY(0px)")
      expect(background?.style.height).toBe("36px")
      expect(container.querySelector(".sidebar-nav-indicator")).toBe(background)
    } finally {
      restoreMetrics()
    }
  })
})
