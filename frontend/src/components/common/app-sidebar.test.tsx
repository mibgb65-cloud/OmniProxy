import { createMemoryHistory, createRouter, RouterContextProvider } from "@tanstack/react-router"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it } from "vitest"
import { routeTree } from "../../app/routeTree.gen"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { AppSidebar } from "./app-sidebar"

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
  return render(<AppSidebar />, { wrapper: Wrapper })
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
  })

  it("switches the whole navigation to English", async () => {
    const user = userEvent.setup()
    renderSidebar()

    await user.click(screen.getByRole("button", { name: "English" }))

    await waitFor(() => {
      expect(screen.getByText("Dashboard")).toBeInTheDocument()
    })
    expect(screen.getByText("Accounts")).toBeInTheDocument()
    expect(screen.queryByText("仪表盘")).not.toBeInTheDocument()
  })

  it("collapses to icons only and restores labels", async () => {
    const user = userEvent.setup()
    renderSidebar()

    await user.click(screen.getByRole("button", { name: "折叠" }))
    expect(screen.queryByText("仪表盘")).not.toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "展开" }))
    expect(screen.getByText("仪表盘")).toBeInTheDocument()
  })
})
