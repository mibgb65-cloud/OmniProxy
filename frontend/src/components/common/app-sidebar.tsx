import { Tooltip } from "../ui/tooltip"
import { Link } from "@tanstack/react-router"
import {
  BarChart3,
  Gauge,
  History,
  LayoutDashboard,
  Network,
  PanelLeftClose,
  PanelLeftOpen,
  Route as RouteIcon,
  ScrollText,
  Server,
  Users,
} from "lucide-react"
import { useTranslation } from "react-i18next"
import { SidebarLabel } from "./sidebar-label"
import { SidebarGatewayStatus } from "../../features/gateway/sidebar-gateway-status"
import { AppPreferencesDialog } from "../../features/settings/app-preferences-dialog"
import { ScrollRegion } from "./scroll-region"
import { cn } from "../../lib/utils"
import { useUIStore } from "../../stores/ui-store"
import type { NavItem } from "../../types/ui"
import "./app-sidebar.css"

const navItems: NavItem[] = [
  { to: "/", labelKey: "nav.dashboard", icon: LayoutDashboard },
  { to: "/accounts", labelKey: "nav.accounts", icon: Users },
  { to: "/providers", labelKey: "nav.providers", icon: Server },
  { to: "/gateway", labelKey: "nav.gateway", icon: Network },
  { to: "/routing", labelKey: "nav.routing", icon: RouteIcon },
  { to: "/quota", labelKey: "nav.quota", icon: Gauge },
  { to: "/usage", labelKey: "nav.usage", icon: BarChart3 },
  { to: "/requests", labelKey: "nav.requests", icon: History },
  { to: "/logs", labelKey: "nav.logs", icon: ScrollText },
]

export function AppSidebar() {
  const { t } = useTranslation()
  const collapsed = useUIStore((s) => s.sidebarCollapsed)
  const toggleSidebar = useUIStore((s) => s.toggleSidebar)

  return (
    <aside
      className={cn(
        "app-sidebar border-sidebar-border bg-sidebar text-sidebar-foreground flex h-full shrink-0 flex-col overflow-hidden border-r",
        collapsed ? "w-16" : "w-56",
      )}
    >
      {/* 品牌栏与标题栏同高，横向分隔线连续；全窗口只在这里出现品牌标识。 */}
      <div
        className="border-sidebar-border flex h-11 shrink-0 items-center gap-2 overflow-hidden border-b"
        data-testid="sidebar-brand"
      >
        {collapsed ? (
          <Tooltip content={t("common.expand")} side="right">
            <button
              type="button"
              aria-label={t("common.expand")}
              aria-expanded={false}
              onClick={toggleSidebar}
              className="group focus-visible:ring-ring relative flex h-11 w-full cursor-pointer items-center justify-center outline-none focus-visible:ring-2 focus-visible:ring-inset"
            >
              {/* 收起态整行就是展开按钮：默认显示品牌图标，悬停／聚焦换成展开图标。 */}
              <img
                src="/app-icon.png"
                alt=""
                draggable={false}
                className="size-6 shrink-0 transition-opacity select-none group-hover:opacity-0 group-focus-visible:opacity-0"
              />
              <PanelLeftOpen className="absolute size-4 opacity-0 transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100" />
            </button>
          </Tooltip>
        ) : (
          <>
            <img
              src="/app-icon.png"
              alt=""
              draggable={false}
              className="ml-3 size-6 shrink-0 select-none"
            />
            <SidebarLabel collapsed={collapsed} className="font-semibold">
              OmniProxy
            </SidebarLabel>
            <Tooltip content={t("common.collapse")}>
              <button
                type="button"
                aria-label={t("common.collapse")}
                aria-expanded
                onClick={toggleSidebar}
                className="text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-ring mr-2 ml-auto flex size-8 shrink-0 cursor-pointer items-center justify-center rounded-md outline-none focus-visible:ring-2 focus-visible:ring-inset"
              >
                <PanelLeftClose className="size-4" />
              </button>
            </Tooltip>
          </>
        )}
      </div>

      <SidebarGatewayStatus collapsed={collapsed} />

      <ScrollRegion
        as="nav"
        className="scroll-region--overlay min-h-0 min-w-0 flex-1"
        viewportClassName="p-2"
        ariaLabel="Primary"
      >
        <ul className="space-y-1">
          {navItems.map((item) => (
            <li key={item.to}>
              <SidebarLink item={item} collapsed={collapsed} label={t(item.labelKey)} />
            </li>
          ))}
        </ul>
      </ScrollRegion>

      <div className="border-sidebar-border flex shrink-0 flex-col gap-2 overflow-hidden border-t p-2 whitespace-nowrap">
        <AppPreferencesDialog collapsed={collapsed} />
      </div>
    </aside>
  )
}

function SidebarLink({
  item,
  collapsed,
  label,
}: {
  item: NavItem
  collapsed: boolean
  label: string
}) {
  const Icon = item.icon
  return (
    <Tooltip content={collapsed ? label : undefined} side="right">
      <Link
        to={item.to}
        aria-label={label}
        // 桌面导航不提供网页链接拖放，避免误拖时出现标题与本地 URL 的拖影。
        draggable={false}
        onDragStart={(event) => event.preventDefault()}
        // 固定行高，防止移除文字后由较矮的图标撑高而导致整列导航上移。
        className="hover:bg-sidebar-accent hover:text-sidebar-accent-foreground data-[active=true]:bg-sidebar-accent data-[active=true]:text-sidebar-accent-foreground flex h-9 items-center gap-2 overflow-hidden rounded-md px-4 text-sm whitespace-nowrap select-none data-[active=true]:font-medium"
        activeProps={{ "data-active": "true" }}
      >
        <Icon className="size-4 shrink-0" />
        <SidebarLabel collapsed={collapsed}>{label}</SidebarLabel>
      </Link>
    </Tooltip>
  )
}
