import { createRootRoute, Outlet } from "@tanstack/react-router"
import { AppSidebar } from "../../components/common/app-sidebar"
import { AppTitlebar } from "../../components/common/app-titlebar"

export const Route = createRootRoute({
  component: RootLayout,
})

function RootLayout() {
  return (
    <div className="bg-background text-foreground flex h-screen w-screen flex-col overflow-hidden">
      <AppTitlebar />
      <div className="flex min-h-0 flex-1">
        <AppSidebar />
        <main className="min-w-0 flex-1 overflow-y-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
