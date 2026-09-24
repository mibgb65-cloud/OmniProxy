import { createRootRoute, Outlet } from "@tanstack/react-router"
import { AppSidebar } from "../../components/common/app-sidebar"

export const Route = createRootRoute({
  component: RootLayout,
})

function RootLayout() {
  return (
    <div className="bg-background text-foreground flex h-screen w-screen overflow-hidden">
      <AppSidebar />
      <main className="flex-1 overflow-y-auto p-6">
        <Outlet />
      </main>
    </div>
  )
}
