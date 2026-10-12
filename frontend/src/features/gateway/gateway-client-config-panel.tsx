import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { GatewayClientCopyButton } from "./gateway-client-copy-button"
import { getGatewayClientConfig, type GatewayClient } from "./gateway-client-configuration"

export function GatewayClientConfigPanel({
  client,
  baseUrl,
  name,
}: {
  client: GatewayClient
  baseUrl: string | null
  name: string
}) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<"cli" | "desktop">("cli")
  const config = baseUrl ? getGatewayClientConfig(client, baseUrl) : null
  const claudeDesktop = client === "claude" && mode === "desktop"

  return (
    <div className="min-w-0 space-y-3 border-t pt-4">
      <div
        role="group"
        aria-label={t("gateway.clients.mode", { client: name })}
        className="flex gap-2"
      >
        {(["cli", "desktop"] as const).map((item) => (
          <Button
            key={item}
            type="button"
            size="sm"
            variant={mode === item ? "secondary" : "outline"}
            aria-pressed={mode === item}
            onClick={() => setMode(item)}
          >
            {t(item === "cli" ? "gateway.clients.cli" : "gateway.clients.desktop")}
          </Button>
        ))}
      </div>
      {claudeDesktop ? (
        <>
          <ol className="list-decimal space-y-2 pl-5 text-sm leading-6">
            <li>{t("gateway.clients.claude.desktop.developer")}</li>
            <li>{t("gateway.clients.claude.desktop.configure")}</li>
            <li>{t("gateway.clients.claude.desktop.address")}</li>
          </ol>
          <p className="text-muted-foreground text-xs leading-5">
            {t("gateway.clients.claude.desktop.note")}
          </p>
        </>
      ) : (
        <>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <code className="text-xs">
              {client === "codex" ? "~/.codex/config.toml" : "~/.claude/settings.json"}
            </code>
            <GatewayClientCopyButton
              value={config}
              label={t("gateway.clients.copyConfig", { client: name })}
            />
          </div>
          {config ? (
            <pre className="bg-muted/30 min-w-0 rounded-lg border p-3 text-xs leading-6 wrap-anywhere whitespace-pre-wrap">
              <code>{config}</code>
            </pre>
          ) : (
            <p className="text-muted-foreground text-sm">
              {t("gateway.clients.configurationUnavailable")}
            </p>
          )}
          <p className="text-muted-foreground text-xs leading-5">
            {t(
              client === "codex"
                ? mode === "desktop"
                  ? "gateway.clients.codex.desktop"
                  : "gateway.clients.codex.cli"
                : "gateway.clients.claude.cli",
            )}
          </p>
        </>
      )}
    </div>
  )
}
