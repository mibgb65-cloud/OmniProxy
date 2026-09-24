import { Link } from "@tanstack/react-router"
import {
  BarChart3,
  Gauge,
  History,
  LayoutDashboard,
  Network,
  PanelLeftClose,
  Route as RouteIcon,
  ScrollText,
  Server,
  Settings,
  Users,
} from "lucide-react"
import { Button } from "../ui/button"
import { LanguageSwitcher } from "./language-switcher"
import { useTranslation } from "../../hooks/use-translation"
import { cn } from "../../lib/utils"
import { useUIStore } from "../../stores/ui-store"
import type { NavItem } from "../../types/ui"

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
]

export function AppSidebar() {
  const { t } = useTranslation()
  const collapsed = useUIStore((s) => s.sidebarCollapsed)
  const toggleSidebar = useUIStore((s) => s.toggleSidebar)

  return (
    <aside
      className={cn(
        "border-sidebar-border bg-sidebar text-sidebar-foreground flex h-full flex-col border-r transition-[width]",
        collapsed ? "w-16" : "w-56",
      )}
    >
      <div className="border-sidebar-border flex h-14 items-center gap-2 border-b px-4">
        <span className="bg-primary size-6 shrink-0 rounded-md" aria-hidden="true" />
        {collapsed ? null : <span className="font-semibold">OmniProxy</span>}
      </div>

      <nav className="flex-1 overflow-y-auto p-2" aria-label="Primary">
        <ul className="space-y-1">
          {navItems.map((item) => (
            <li key={item.to}>
              <SidebarLink item={item} collapsed={collapsed} label={t(item.labelKey)} />
            </li>
          ))}
        </ul>
      </nav>

      <div className="border-sidebar-border space-y-2 border-t p-2">
        <LanguageSwitcher collapsed={collapsed} />
        <Button
          variant="ghost"
          size="sm"
          className="w-full justify-start gap-2"
          onClick={toggleSidebar}
          aria-label={collapsed ? t("common.expand") : t("common.collapse")}
        >
          <PanelLeftClose className={cn("size-4 shrink-0", collapsed && "rotate-180")} />
          {collapsed ? null : <span>{t("common.collapse")}</span>}
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
    <Link
      to={item.to}
      title={collapsed ? label : undefined}
      className="hover:bg-sidebar-accent hover:text-sidebar-accent-foreground data-[active=true]:bg-sidebar-accent data-[active=true]:text-sidebar-accent-foreground flex items-center gap-2 rounded-md px-3 py-2 text-sm data-[active=true]:font-medium"
      activeProps={{ "data-active": "true" }}
    >
      <Icon className="size-4 shrink-0" />
      {collapsed ? null : <span>{label}</span>}
    </Link>
  )
}
