import { createFileRoute } from "@tanstack/react-router"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card"
import { PagePlaceholder } from "../../components/common/page-placeholder"
import { useAppInfo } from "../../hooks/use-app-info"
import { usePing } from "../../hooks/use-ping"
import { useTranslation } from "../../hooks/use-translation"

export const Route = createFileRoute("/")({
  component: DashboardPage,
})

function DashboardPage() {
  const { t } = useTranslation()
  const { data, isPending, isError, error } = useAppInfo()
  const pingMutation = usePing()

  return (
    <PagePlaceholder titleKey="dashboard.title" descriptionKey="dashboard.description">
      <div className="grid gap-4 sm:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>{t("dashboard.application")}</CardTitle>
            <CardDescription>{t("dashboard.applicationDescription")}</CardDescription>
          </CardHeader>
          <CardContent className="text-sm">
            {isPending ? <p className="text-muted-foreground">{t("dashboard.loading")}</p> : null}
            {isError ? (
              <p className="text-destructive">
                {t("dashboard.loadFailed", { error: error.message })}
              </p>
            ) : null}
            {data ? (
              <dl className="grid grid-cols-2 gap-y-2">
                <dt className="text-muted-foreground">{t("dashboard.version")}</dt>
                <dd className="text-right">{data.version}</dd>
                <dt className="text-muted-foreground">{t("dashboard.profile")}</dt>
                <dd className="text-right">{data.profile}</dd>
                <dt className="text-muted-foreground">{t("dashboard.proxyPort")}</dt>
                <dd className="text-right">{data.proxyPort}</dd>
                <dt className="text-muted-foreground">{t("dashboard.controlPort")}</dt>
                <dd className="text-right">{data.controlPort}</dd>
                <dt className="text-muted-foreground">{t("dashboard.dataDirectory")}</dt>
                <dd className="truncate text-right" title={data.dataDir}>
                  {data.dataDir}
                </dd>
              </dl>
            ) : null}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{t("dashboard.bindingCheck")}</CardTitle>
            <CardDescription>{t("dashboard.bindingCheckDescription")}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <Button
              size="sm"
              onClick={() => pingMutation.mutate()}
              disabled={pingMutation.isPending}
            >
              {pingMutation.isPending ? t("dashboard.pinging") : t("dashboard.ping")}
            </Button>
            {pingMutation.isError ? (
              <p className="text-destructive">
                {t("dashboard.pingFailed", { error: pingMutation.error.message })}
              </p>
            ) : null}
            {pingMutation.isSuccess ? (
              <p className="text-muted-foreground">
                {t("dashboard.pingReply", { result: pingMutation.data })}
              </p>
            ) : null}
          </CardContent>
        </Card>
      </div>
    </PagePlaceholder>
  )
}
