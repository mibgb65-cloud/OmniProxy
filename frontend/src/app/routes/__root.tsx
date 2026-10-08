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
        <ScrollRegion as="main" className="scroll-region--window-edge min-w-0 flex-1">
          {/* 统一限制页面左右留白，超宽窗口也不会让内容缩在中央。 */}
          <div className="w-full min-w-0 px-[clamp(16px,2.5vw,96px)] py-6">
            <Outlet />
          </div>
        </ScrollRegion>
      </div>
    </div>
  )
}
