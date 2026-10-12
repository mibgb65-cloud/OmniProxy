import { createRootRoute, Outlet, useMatches } from "@tanstack/react-router"
import { AppSidebar } from "../../components/common/app-sidebar"
import { AppTitlebar } from "../../components/common/app-titlebar"
import { ScrollRegion } from "../../components/common/scroll-region"

export const Route = createRootRoute({
  component: RootLayout,
})

function RootLayout() {
  // 地址会先于 Outlet 更新；跟随已提交的匹配，避免用上一页高度恢复新页面。
  const pageKey = useMatches({ select: (matches) => matches.at(-1)?.pathname })
  return (
    <div className="bg-background text-foreground flex h-full w-full flex-col overflow-hidden">
      <AppTitlebar />
      <div className="flex min-h-0 min-w-0 flex-1 overflow-hidden">
        <AppSidebar />
        <ScrollRegion
          as="main"
          pageKey={pageKey}
          className="scroll-region--window-edge min-w-0 flex-1"
        >
          {/* 统一限制页面左右留白，超宽窗口也不会让内容缩在中央。 */}
          <div className="w-full min-w-0 px-[clamp(16px,2.5vw,96px)] py-6">
            <Outlet />
          </div>
        </ScrollRegion>
      </div>
    </div>
  )
}
