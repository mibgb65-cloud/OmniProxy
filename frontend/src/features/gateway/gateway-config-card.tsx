import { useMutation } from "@tanstack/react-query"
import { Link } from "@tanstack/react-router"
import { Clipboard } from "@wailsio/runtime"
import { Check, Copy, Terminal } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import type { GatewayPreparation } from "./use-gateway-preparation"

export function GatewayConfigCard({ data }: { data: GatewayPreparation }) {
  const { t } = useTranslation()
  const copy = useMutation({ mutationFn: (address: string) => Clipboard.SetText(address) })
  const copied = copy.isSuccess && copy.variables === data.address
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
    {
      label: t("gateway.summary.protocol"),
      value: data.gateway.info.isSuccess ? data.gateway.info.data.protocol : "—",
    },
    {
      label: t("gateway.summary.providers"),
      value:
        data.providerCount ??
        t(data.providerState === "loading" ? "gateway.loading" : "gateway.unavailable"),
    },
    {
      label: t("gateway.summary.accounts"),
      value:
        data.accountCount ??
        t(data.accountState === "loading" ? "gateway.loading" : "gateway.unavailable"),
    },
    { label: t("gateway.config.profile"), value: data.runtime.isSuccess ? profileLabel : "—" },
  ]
  return (
    <Card className="min-w-0 gap-4 py-4">
      <CardHeader className="px-4">
        <h2 className="text-base font-semibold">{t("gateway.overview.connection")}</h2>
      </CardHeader>
      <CardContent className="space-y-3 px-4 text-sm">
        <div className="bg-muted/25 flex min-w-0 items-center gap-2 rounded-lg border p-3">
          <div className="min-w-0 flex-1 space-y-1">
            <p className="text-muted-foreground text-xs">{t("gateway.overview.baseUrl")}</p>
            <code className="block text-xs wrap-anywhere">
              {data.address ??
                t(data.runtime.isPending ? "gateway.loading" : "gateway.unavailable")}
            </code>
          </div>
          <Button
            variant="ghost"
            size="icon-sm"
            disabled={!data.address || copy.isPending}
            aria-label={t("gateway.connection.copy")}
            onClick={() => data.address && copy.mutate(data.address)}
          >
            {copied ? <Check aria-hidden="true" /> : <Copy aria-hidden="true" />}
          </Button>
        </div>
        {copied ? (
          <p role="status" className="text-muted-foreground text-xs">
            {t("gateway.connection.copied")}
          </p>
        ) : null}
        {copy.isError ? (
          <p role="alert" className="text-destructive text-xs">
            {t("gateway.connection.copyFailed")}
          </p>
        ) : null}
        <dl className="divide-y">
          {rows.map(({ label, value }) => (
            <div
              key={label}
              className="flex flex-wrap items-center justify-between gap-2 py-2.5 text-xs"
            >
              <dt className="text-muted-foreground">{label}</dt>
              <dd className="font-medium tabular-nums">{value}</dd>
            </div>
          ))}
        </dl>
        <Button asChild variant="outline" size="sm" className="w-full">
          <Link to="/gateway" search={{ tab: "clients" }}>
            <Terminal aria-hidden="true" />
            {t("gateway.tabs.clients")}
          </Link>
        </Button>
      </CardContent>
    </Card>
  )
}
