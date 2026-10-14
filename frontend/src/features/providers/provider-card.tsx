import { Tooltip } from "../../components/ui/tooltip"
import type { ReactNode } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent } from "../../components/ui/card"
import type { ProviderProbeResult } from "../../services/omniproxy"
import { ProviderBrandIcon } from "./provider-brand-icon"
import { ProviderPowerButton } from "./provider-power-button"
import { ModelPreview } from "./model-preview"
import "./provider-probe-tooltip.css"

export type ProbeDisplay = { result?: ProviderProbeResult; error?: boolean; visible?: boolean }

export function ProviderCard({
  kind,
  name,
  enabled,
  disabled,
  authMethods,
  plannedAuthMethods,
  models,
  endpoint,
  probeResult,
  probeError,
  probeVisible,
  probePending,
  onToggle,
  onProbe,
  actions,
}: {
  kind: string
  name: string
  enabled: boolean
  disabled: boolean
  authMethods: string[] | null
  plannedAuthMethods: string[] | null
  models: string[]
  endpoint?: string
  probeResult?: ProviderProbeResult
  probeError?: boolean
  probeVisible?: boolean
  probePending: boolean
  onToggle: (enabled: boolean) => void
  onProbe: () => void
  actions?: ReactNode
}) {
  const { t } = useTranslation()
  return (
    <Card className="dark:bg-card/75 gap-0 py-0 shadow-sm">
      <CardContent className="space-y-3 px-4 py-4">
        <div className="flex items-start justify-between gap-3">
          <div className="flex min-w-0 items-start gap-3">
            <ProviderBrandIcon kind={kind} endpoint={endpoint} />
            <div className="min-w-0 space-y-1">
              <Tooltip content={name}>
                <h3 className="truncate text-sm font-semibold">{name}</h3>
              </Tooltip>
              <ModelPreview name={name} models={models} />
              {endpoint ? (
                <Tooltip content={endpoint}>
                  <p className="text-muted-foreground truncate text-xs">{endpoint}</p>
                </Tooltip>
              ) : null}
            </div>
          </div>
          <ProviderPowerButton
            name={name}
            enabled={enabled}
            disabled={disabled}
            onChange={onToggle}
          />
        </div>
        <div className="flex flex-wrap gap-1.5">
          {authMethods?.includes("api_key") ? (
            <span className="border-border bg-muted/50 rounded-md border px-2 py-0.5 text-xs">
              {t("providers.apiKey")}
            </span>
          ) : null}
          {plannedAuthMethods?.includes("subscription_login") ? (
            <span className="rounded-md border border-blue-200 bg-blue-50 px-2 py-0.5 text-xs text-blue-700 dark:border-blue-500/30 dark:bg-blue-500/10 dark:text-blue-300">
              {t("providers.subscriptionPlanned")}
            </span>
          ) : null}
          {authMethods?.includes("subscription_login") ? (
            <span className="border-border bg-muted/50 rounded-md border px-2 py-0.5 text-xs">
              {t("subscription.type")}
            </span>
          ) : null}
        </div>
        <div className="border-border flex flex-wrap items-center justify-between gap-2 border-t pt-3">
          <span className="flex items-center gap-1.5 text-xs">
            <span
              className={`size-1.5 rounded-full ${enabled ? "bg-emerald-500" : "bg-muted-foreground"}`}
              aria-hidden="true"
            />
            {enabled ? t("providers.enabled") : t("providers.disabled")}
          </span>
          <div className="flex flex-wrap items-center gap-1">
            <div className="relative">
              {probeResult || probeError ? (
                <span
                  role="status"
                  aria-hidden={!probeVisible}
                  data-state={probeVisible ? "open" : "closed"}
                  className={`provider-probe-tip bg-popover pointer-events-none absolute top-1/2 right-full z-20 mr-2 w-max max-w-[200px] rounded-md border px-2.5 py-1.5 text-xs leading-tight shadow-md ${probeResult?.reachable ? "border-emerald-500/30 text-emerald-700 dark:text-emerald-300" : "border-destructive/30 text-destructive"}`}
                >
                  {probeError
                    ? t("providers.testError")
                    : probeResult?.reachable
                      ? t("providers.testReachable", {
                          latency: probeResult.latencyMs,
                          status: probeResult.statusCode,
                        })
                      : t("providers.testUnreachable", { latency: probeResult?.latencyMs ?? 0 })}
                </span>
              ) : null}
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={onProbe}
                disabled={probePending || !endpoint}
                className="cursor-pointer"
              >
                {probePending ? t("providers.testing") : t("providers.test")}
              </Button>
            </div>
            {actions}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
