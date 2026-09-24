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
import { cn } from "../../lib/utils"
import { useUIStore } from "../../stores/ui-store"
import type { NavItem } from "../../types/ui"

const navItems: NavItem[] = [
  { to: "/", label: "Dashboard", icon: LayoutDashboard },
  { to: "/accounts", label: "Accounts", icon: Users },
  { to: "/providers", label: "Providers", icon: Server },
  { to: "/gateway", label: "Gateway", icon: Network },
  { to: "/routing", label: "Routing", icon: RouteIcon },
  { to: "/quota", label: "Quota", icon: Gauge },
  { to: "/usage", label: "Usage", icon: BarChart3 },
  { to: "/requests", label: "Requests", icon: History },
  { to: "/logs", label: "Logs", icon: ScrollText },
  { to: "/settings", label: "Settings", icon: Settings },
]

export function AppSidebar() {
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
              <SidebarLink item={item} collapsed={collapsed} />
            </li>
          ))}
        </ul>
      </nav>

      <div className="border-sidebar-border border-t p-2">
        <Button
          variant="ghost"
          size="sm"
          className="w-full justify-start gap-2"
          onClick={toggleSidebar}
          aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
        >
          <PanelLeftClose className={cn("size-4 shrink-0", collapsed && "rotate-180")} />
          {collapsed ? null : <span>Collapse</span>}
        </Button>
      </div>
    </aside>
  )
}

function SidebarLink({ item, collapsed }: { item: NavItem; collapsed: boolean }) {
  const Icon = item.icon
  return (
    <Link
      to={item.to}
      title={collapsed ? item.label : undefined}
      className="hover:bg-sidebar-accent hover:text-sidebar-accent-foreground data-[active=true]:bg-sidebar-accent data-[active=true]:text-sidebar-accent-foreground flex items-center gap-2 rounded-md px-3 py-2 text-sm data-[active=true]:font-medium"
      activeProps={{ "data-active": "true" }}
    >
      <Icon className="size-4 shrink-0" />
      {collapsed ? null : <span>{item.label}</span>}
    </Link>
  )
}
