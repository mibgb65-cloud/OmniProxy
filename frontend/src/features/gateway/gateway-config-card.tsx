import { useTranslation } from "react-i18next"
import { Card, CardContent, CardDescription, CardHeader } from "../../components/ui/card"
import type { GatewayPreparation } from "./use-gateway-preparation"

export function GatewayConfigCard({ data }: { data: GatewayPreparation }) {
  const { t } = useTranslation()
  const profile = data.runtime.data?.profile
  const profileLabel =
    profile === "production"
      ? t("gateway.config.profile.production")
      : profile === "development"
        ? t("gateway.config.profile.development")
        : profile === "test"
          ? t("gateway.config.profile.test")
          : t("gateway.config.profile.unknown")
  const rows = [
    { label: t("gateway.config.host"), value: "127.0.0.1" },
    { label: t("gateway.config.scope"), value: t("gateway.config.localOnly") },
    {
      label: t("gateway.config.port"),
      value: data.address
        ? data.gateway.isRunning
          ? data.gateway.info.data?.runtime.port
          : data.runtime.data?.proxyPort
        : "—",
    },
    { label: t("gateway.config.profile"), value: data.runtime.isSuccess ? profileLabel : "—" },
  ]

  return (
    <Card className="dark:bg-card/75 min-w-0 gap-4">
      <CardHeader>
        <h2 className="text-base font-semibold">{t("gateway.config.title")}</h2>
        <CardDescription>{t("gateway.config.description")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-1 flex-col justify-between gap-4 text-sm">
        <dl className="divide-border divide-y">
          {rows.map(({ label, value }) => (
            <div key={label} className="flex flex-wrap justify-between gap-2 py-2.5 first:pt-0">
              <dt className="text-muted-foreground">{label}</dt>
              <dd className="font-medium tabular-nums">{value}</dd>
            </div>
          ))}
        </dl>
        <p className="text-muted-foreground text-xs leading-5">{t("gateway.config.readOnly")}</p>
      </CardContent>
    </Card>
  )
}
