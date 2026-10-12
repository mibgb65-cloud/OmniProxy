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
    <Card className="min-w-0 py-4">
      <CardContent className="flex flex-wrap items-center justify-between gap-4 px-4">
        <h2 className="text-sm font-semibold">{t("dash.clients")}</h2>
        <div className="flex min-w-0 basis-full flex-wrap gap-3 xl:flex-1 xl:basis-0">
          {failed ? (
            <p role="alert" className="text-destructive text-xs">
              {t("dash.clientsFailed")}
            </p>
          ) : items.length ? (
            items.map((item) => (
              <span
                key={item.client}
                className="bg-muted/40 inline-flex max-w-full min-w-0 flex-wrap items-center gap-2 rounded-lg border px-3 py-2 text-xs"
              >
                <span className="font-medium">{t(agentTitle(item.client))}</span>
                <span className="text-muted-foreground">
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
              </span>
            ))
          ) : (
            <span className="text-muted-foreground text-xs">
              {t(loading ? "accounts.loading" : "dash.clientsEmpty")}
            </span>
          )}
        </div>
        <Link
          to="/gateway"
          className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-xs"
        >
          {t("dash.configureClients")}
          <ArrowUpRight className="size-3" aria-hidden="true" />
        </Link>
      </CardContent>
    </Card>
  )
}
