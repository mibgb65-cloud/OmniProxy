import { Link } from "@tanstack/react-router"
import { ArrowUpRight } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card, CardContent } from "../../components/ui/card"
import { agentTitle, type AgentClient } from "../../services/agents"

export function DashboardClients({
  items,
  failed,
  loading,
}: {
  items: AgentClient[]
  failed: boolean
  loading: boolean
}) {
  const { t } = useTranslation()
  return (
    <Card className="flex h-full min-w-0 flex-col gap-3 py-4">
      <CardContent className="flex min-h-0 flex-1 flex-col gap-3 px-4">
        <div className="flex items-start justify-between gap-2">
          <h2 className="text-base font-semibold">{t("dash.clients")}</h2>
          <Link
            to="/gateway"
            className="text-muted-foreground hover:text-foreground inline-flex shrink-0 items-center gap-1 text-xs"
          >
            {t("dash.configureClients")}
            <ArrowUpRight className="size-3" aria-hidden="true" />
          </Link>
        </div>
        {failed ? (
          <p role="alert" className="text-destructive text-xs">
            {t("dash.clientsFailed")}
          </p>
        ) : items.length ? (
          <ul className="flex min-w-0 flex-wrap content-start gap-2">
            {items.map((item) => (
              <li
                key={item.client}
                className="bg-muted/40 inline-flex max-w-full min-w-0 items-center gap-2 rounded-lg border px-2.5 py-1.5 text-xs"
              >
                <span className="truncate font-medium">{t(agentTitle(item.client))}</span>
                <span className="text-muted-foreground shrink-0">
                  {t(
                    item.errorCode === "key_disabled"
                      ? "dash.clientsDisabled"
                      : item.errorCode
                        ? "dash.clientsConflict"
                        : item.configured
                          ? "dash.clientsConfigured"
                          : "dash.clientsMissing",
                  )}
                </span>
              </li>
            ))}
          </ul>
        ) : (
          <span className="text-muted-foreground text-xs">
            {t(loading ? "accounts.loading" : "dash.clientsEmpty")}
          </span>
        )}
      </CardContent>
    </Card>
  )
}
