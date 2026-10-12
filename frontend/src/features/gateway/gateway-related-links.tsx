import { Link } from "@tanstack/react-router"
import { ArrowUpRight, History, Route, ScrollText } from "lucide-react"
import { useTranslation } from "react-i18next"

const relatedPages = [
  {
    to: "/routing",
    title: "gateway.links.routing",
    hint: "gateway.links.routingHint",
    icon: Route,
  },
  {
    to: "/requests",
    title: "gateway.links.requests",
    hint: "gateway.links.requestsHint",
    icon: History,
  },
  { to: "/logs", title: "gateway.links.logs", hint: "gateway.links.logsHint", icon: ScrollText },
] as const

export function GatewayRelatedLinks({ diagnostics = false }: { diagnostics?: boolean }) {
  const { t } = useTranslation()
  const pages = relatedPages.filter(({ to }) =>
    diagnostics ? to !== "/routing" : to === "/routing",
  )
  return (
    <nav aria-label={t("gateway.links.title")} className="grid gap-3 sm:grid-cols-2">
      {pages.map(({ to, title, hint, icon: Icon }) => (
        <Link
          key={to}
          to={to}
          className="hover:bg-muted/40 focus-visible:ring-ring/50 flex min-w-0 items-center gap-3 rounded-lg border p-4 transition-colors focus-visible:ring-2 motion-reduce:transition-none"
        >
          <Icon className="text-muted-foreground size-4 shrink-0" aria-hidden="true" />
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium">{t(title)}</p>
            <p className="text-muted-foreground mt-1 text-xs leading-5">{t(hint)}</p>
          </div>
          <ArrowUpRight className="text-muted-foreground size-4 shrink-0" aria-hidden="true" />
        </Link>
      ))}
    </nav>
  )
}
