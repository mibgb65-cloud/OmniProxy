import { useTranslation } from "react-i18next"
import { Card, CardContent, CardDescription, CardHeader } from "../../components/ui/card"
import type { GatewayPreparation } from "./use-gateway-preparation"

export function GatewayStatusCard({ data }: { data: GatewayPreparation }) {
  const { t } = useTranslation()
  const metrics = [
    { label: t("gateway.summary.providers"), value: data.providerCount, state: data.providerState },
    { label: t("gateway.summary.accounts"), value: data.accountCount, state: data.accountState },
  ]

  return (
    <Card className="dark:bg-card/75 min-w-0 gap-4">
      <CardHeader>
        <h2 className="text-base font-semibold">{t("gateway.summary.title")}</h2>
        <CardDescription>{t("gateway.status.description")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <dl className="grid gap-5 sm:grid-cols-2">
          {metrics.map(({ label, value, state }) => (
            <div key={label}>
              <dt className="text-muted-foreground text-xs">{label}</dt>
              <dd className="mt-2 text-2xl font-semibold tabular-nums">
                {value !== null ? (
                  value
                ) : (
                  <span className="text-muted-foreground text-sm font-normal">
                    {t(state === "loading" ? "gateway.loading" : "gateway.unavailable")}
                  </span>
                )}
              </dd>
            </div>
          ))}
          <div className="border-t pt-4 sm:col-span-2">
            <dt className="text-muted-foreground text-xs">{t("gateway.summary.protocol")}</dt>
            <dd className="mt-2 text-sm">
              {data.gateway.info.isSuccess
                ? data.gateway.info.data.protocol
                : t(data.gateway.info.isPending ? "gateway.loading" : "gateway.unavailable")}
            </dd>
          </div>
        </dl>
        <p className="text-muted-foreground text-xs leading-5">
          {t("gateway.summary.accountsHint")}
        </p>
      </CardContent>
    </Card>
  )
}
