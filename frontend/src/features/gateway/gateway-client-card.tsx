import { useId } from "react"
import { ExternalLink, Terminal } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardDescription, CardHeader } from "../../components/ui/card"
import { Input } from "../../components/ui/input"
import { GatewayClientConfigPanel } from "./gateway-client-config-panel"
import { GatewayClientCopyButton } from "./gateway-client-copy-button"
import {
  gatewayClientDefinitions,
  getGatewayClientConnection,
  type GatewayClient,
} from "./gateway-client-configuration"
import type { GatewayPreparation } from "./use-gateway-preparation"

export function GatewayClientCard({
  client,
  data,
}: {
  client: GatewayClient
  data: GatewayPreparation
}) {
  const { t } = useTranslation()
  const id = useId()
  const definition = gatewayClientDefinitions[client]
  const name = client === "codex" ? "Codex" : "Claude"
  const connection = getGatewayClientConnection(client, data.address)

  return (
    <Card role="region" aria-labelledby={`${id}-title`} className="dark:bg-card/75 min-w-0 gap-4">
      <CardHeader>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 id={`${id}-title`} className="flex items-center gap-2 text-base font-semibold">
            <Terminal className="size-4" aria-hidden="true" />
            {t(definition.title)}
          </h2>
          <span className="bg-muted text-muted-foreground rounded-full border px-2.5 py-0.5 text-xs">
            {t("gateway.clients.pending")}
          </span>
        </div>
        <CardDescription>{t(definition.description)}</CardDescription>
      </CardHeader>
      <CardContent className="flex min-w-0 flex-1 flex-col gap-4">
        <dl className="space-y-3 text-sm">
          <div className="flex flex-wrap justify-between gap-2">
            <dt className="text-muted-foreground">{t("gateway.clients.protocol")}</dt>
            <dd>{definition.protocol}</dd>
          </div>
          <div className="space-y-2">
            <dt>
              <label className="text-muted-foreground text-xs" htmlFor={`${id}-base-url`}>
                {t("gateway.clients.baseUrl")}
              </label>
            </dt>
            <dd className="flex min-w-0 items-start gap-2">
              <Input
                id={`${id}-base-url`}
                readOnly
                className="min-w-0 flex-1 font-mono"
                value={
                  connection?.baseUrl ??
                  (data.runtimeState === "loading" ? t("gateway.loading") : "—")
                }
              />
              <GatewayClientCopyButton
                value={connection?.baseUrl ?? null}
                label={t("gateway.clients.copyAddress", { client: name })}
              />
            </dd>
          </div>
          <div className="space-y-1">
            <dt className="text-muted-foreground text-xs">{t("gateway.clients.endpoint")}</dt>
            <dd className="font-mono text-xs wrap-anywhere">
              {connection ? `POST ${connection.requestUrl}` : "—"}
            </dd>
          </div>
        </dl>
        <p className="text-muted-foreground rounded-lg border border-dashed px-3 py-2 text-xs leading-5">
          {t("gateway.clients.previewNotice")}
        </p>
        <GatewayClientConfigPanel
          client={client}
          baseUrl={connection?.baseUrl ?? null}
          name={name}
        />
        <a
          href={definition.docs}
          target="_blank"
          rel="noopener noreferrer"
          className="text-muted-foreground hover:text-foreground focus-visible:ring-ring/50 mt-auto inline-flex w-fit items-center gap-1.5 rounded text-xs focus-visible:ring-2"
        >
          {t("gateway.clients.docs")}
          <ExternalLink className="size-3" aria-hidden="true" />
        </a>
      </CardContent>
    </Card>
  )
}
