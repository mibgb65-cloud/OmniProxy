import { Tooltip } from "../ui/tooltip"
import { Link } from "@tanstack/react-router"
import {
  BarChart3,
  Gauge,
  History,
  Info,
  LayoutDashboard,
  Network,
  PanelLeftClose,
  Route as RouteIcon,
  ScrollText,
  Server,
  Settings,
  Users,
} from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../ui/button"
import { SidebarLabel } from "./sidebar-label"
import { SidebarGatewayStatus } from "../../features/gateway/sidebar-gateway-status"
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
  { to: "/settings", labelKey: "nav.settings", icon: Settings },
  { to: "/about", labelKey: "nav.about", icon: Info },
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
      {/* 图标固定在收起态的水平中心，避免宽度过渡时切换对齐方式产生跳动。 */}
      <div className="border-sidebar-border flex h-14 shrink-0 items-center gap-2 overflow-hidden border-b px-5 whitespace-nowrap">
        <img src="/app-icon.png" alt="" className="size-6 shrink-0" />
        <SidebarLabel collapsed={collapsed} className="font-semibold">
          OmniProxy
        </SidebarLabel>
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
        <Button
          variant="ghost"
          size="sm"
          className="w-full justify-start gap-2 px-4 transition-colors has-[>svg]:px-4"
          onClick={toggleSidebar}
          aria-label={collapsed ? t("common.expand") : t("common.collapse")}
          aria-expanded={!collapsed}
        >
          <PanelLeftClose className={cn("size-4 shrink-0", collapsed && "rotate-180")} />
          <SidebarLabel collapsed={collapsed}>{t("common.collapse")}</SidebarLabel>
        </Button>
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
        // 固定行高，防止移除文字后由较矮的图标撑高而导致整列导航上移。
        className="hover:bg-sidebar-accent hover:text-sidebar-accent-foreground data-[active=true]:bg-sidebar-accent data-[active=true]:text-sidebar-accent-foreground flex h-9 items-center gap-2 overflow-hidden rounded-md px-4 text-sm whitespace-nowrap data-[active=true]:font-medium"
        activeProps={{ "data-active": "true" }}
      >
        <Icon className="size-4 shrink-0" />
        <SidebarLabel collapsed={collapsed}>{label}</SidebarLabel>
      </Link>
    </Tooltip>
  )
}
