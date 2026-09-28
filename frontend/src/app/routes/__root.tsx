import { createRootRoute, Outlet } from "@tanstack/react-router"
import { AppSidebar } from "../../components/common/app-sidebar"
import { AppTitlebar } from "../../components/common/app-titlebar"
import { ScrollRegion } from "../../components/common/scroll-region"

export const Route = createRootRoute({
  component: RootLayout,
})

function RootLayout() {
  return (
    <div className="bg-background text-foreground flex h-full w-full flex-col overflow-hidden">
      <AppTitlebar />
      <div className="flex min-h-0 min-w-0 flex-1 overflow-hidden">
        <AppSidebar />
        <ScrollRegion as="main" className="min-w-0 flex-1" viewportClassName="p-6">
          <Outlet />
        </ScrollRegion>
      </div>
    </div>
  )
}
