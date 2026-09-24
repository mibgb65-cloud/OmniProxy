import { createMemoryHistory, createRouter, RouterContextProvider } from "@tanstack/react-router"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactNode } from "react"
import { describe, expect, it } from "vitest"
import { routeTree } from "../../app/routeTree.gen"
import { AppSidebar } from "./app-sidebar"

function renderSidebar() {
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: ["/"] }),
  })
  function Wrapper({ children }: { children: ReactNode }) {
    return <RouterContextProvider router={router}>{children}</RouterContextProvider>
  }
  return render(<AppSidebar />, { wrapper: Wrapper })
}

describe("AppSidebar", () => {
  it("renders the primary navigation", () => {
    renderSidebar()
    expect(screen.getByText("OmniProxy")).toBeInTheDocument()
    expect(screen.getByText("Dashboard")).toBeInTheDocument()
    expect(screen.getByText("Accounts")).toBeInTheDocument()
    expect(screen.getByText("Settings")).toBeInTheDocument()
  })

  it("collapses to icons only and restores labels", async () => {
    const user = userEvent.setup()
    renderSidebar()

    await user.click(screen.getByRole("button", { name: "Collapse sidebar" }))
    expect(screen.queryByText("Dashboard")).not.toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "Expand sidebar" }))
    expect(screen.getByText("Dashboard")).toBeInTheDocument()
  })
})
