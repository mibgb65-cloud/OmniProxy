import { createMemoryHistory, createRouter, RouterContextProvider } from "@tanstack/react-router"
import { act, fireEvent, isInaccessible, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { routeTree } from "../../app/routeTree.gen"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { AppSidebar } from "./app-sidebar"

vi.mock("../../features/gateway/sidebar-gateway-status", () => ({
  SidebarGatewayStatus: () => <section aria-label="网关" />,
}))

function renderSidebar() {
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: ["/"] }),
  })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <I18nProvider>
        <RouterContextProvider router={router}>{children}</RouterContextProvider>
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
    expect(screen.getByText("设置")).toBeInTheDocument()
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

  it("collapses to icons only and restores labels", async () => {
    const user = userEvent.setup()
    renderSidebar()

    await user.click(screen.getByRole("button", { name: "折叠" }))
    expect(isInaccessible(screen.getByText("仪表盘"))).toBe(true)
    expect(screen.getByRole("link", { name: "仪表盘" })).toHaveAttribute("href", "/")
    expect(screen.getAllByRole("link")).toHaveLength(11)
    expect(screen.getByRole("button", { name: "展开" })).toHaveAttribute("aria-expanded", "false")

    await user.click(screen.getByRole("button", { name: "展开" }))
    expect(screen.getByText("仪表盘")).toBeInTheDocument()
    expect(isInaccessible(screen.getByText("仪表盘"))).toBe(false)
    expect(screen.getByRole("button", { name: "折叠" })).toHaveAttribute("aria-expanded", "true")
  })
})
