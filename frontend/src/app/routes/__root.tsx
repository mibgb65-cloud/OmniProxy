import { createRootRoute, Outlet, useMatches } from "@tanstack/react-router"
import { AppSidebar } from "../../components/common/app-sidebar"
import { AppTitlebar } from "../../components/common/app-titlebar"
import { ScrollRegion } from "../../components/common/scroll-region"
import { CodexQuotaActivationWatcher } from "../../features/accounts/codex-quota-activation-watcher"

export const Route = createRootRoute({
  component: RootLayout,
})

function RootLayout() {
  // 地址会先于 Outlet 更新；跟随已提交的匹配，避免用上一页高度恢复新页面。
  const pageKey = useMatches({ select: (matches) => matches.at(-1)?.pathname })
  return (
    // 侧栏通顶占据整列高度，品牌标识只留在窗口左上角一次；标题栏只覆盖内容区。
    <div className="bg-background text-foreground flex h-full w-full overflow-hidden">
      <AppSidebar />
      <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        <AppTitlebar />
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
      {/* 后台自动激活的结果在任何页面都提示，不需要先打开 OpenAI 设置面板。 */}
      <CodexQuotaActivationWatcher />
    </div>
  )
}
