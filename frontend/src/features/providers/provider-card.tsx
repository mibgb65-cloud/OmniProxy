import { useId, useState } from "react"
import type { ReactNode } from "react"
import { Bot, ChevronDown } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent } from "../../components/ui/card"
import type { ProviderProbeResult } from "../../services/omniproxy"
import { ProviderSwitch } from "./provider-switch"

const iconColors: Record<string, string> = {
  openai: "bg-zinc-950 text-white dark:bg-zinc-100 dark:text-zinc-900",
  anthropic: "bg-orange-700 text-white",
  google: "bg-blue-600 text-white",
  deepseek: "bg-indigo-600 text-white",
  zhipu: "bg-violet-700 text-white",
  kimi: "bg-purple-700 text-white",
  minimax: "bg-rose-700 text-white",
  mimo: "bg-cyan-700 text-white",
  stepfun: "bg-emerald-700 text-white",
}

export type ProbeDisplay = { result?: ProviderProbeResult; error?: boolean }

function ModelPreview({ models }: { models: string[] }) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const listId = useId()
  if (models.length === 0) {
    return <p className="text-muted-foreground text-xs">{t("providers.customModels")}</p>
  }
  return (
    <div className="min-w-0">
      <button
        type="button"
        aria-expanded={expanded}
        aria-controls={listId}
        onClick={() => setExpanded((value) => !value)}
        className="text-muted-foreground hover:text-foreground flex max-w-full cursor-pointer items-center gap-1 text-left text-xs transition-colors"
      >
        <span className="truncate">{models.slice(0, 2).join(" · ")}</span>
        {models.length > 2 ? <span className="shrink-0">+{models.length - 2}</span> : null}
        <ChevronDown
          className={`size-3.5 shrink-0 motion-safe:transition-transform ${expanded ? "rotate-180" : ""}`}
          aria-hidden="true"
        />
        <span className="sr-only">
          {expanded ? t("providers.hideModels") : t("providers.showModels")}
        </span>
      </button>
      {expanded ? (
        <ul id={listId} className="mt-2 flex flex-wrap gap-1.5">
          {models.map((model) => (
            <li key={model} className="bg-muted/60 rounded-md px-2 py-1 text-xs">
              {model}
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  )
}

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
            <span
              className={`flex size-9 shrink-0 items-center justify-center rounded-lg ${iconColors[kind] ?? "bg-slate-700 text-white"}`}
              aria-hidden="true"
            >
              <Bot className="size-5" />
            </span>
            <div className="min-w-0 space-y-1">
              <h3 className="truncate text-sm font-semibold" title={name}>
                {name}
              </h3>
              <ModelPreview models={models} />
              {endpoint ? (
                <p className="text-muted-foreground truncate text-xs" title={endpoint}>
                  {endpoint}
                </p>
              ) : null}
            </div>
          </div>
          <ProviderSwitch name={name} checked={enabled} disabled={disabled} onChange={onToggle} />
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
            {actions}
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
        </div>
        {probeResult || probeError ? (
          <p
            role="status"
            className={`text-xs ${probeResult?.reachable ? "text-emerald-700 dark:text-emerald-300" : "text-destructive"}`}
          >
            {probeError
              ? t("providers.testError")
              : probeResult?.reachable
                ? t("providers.testReachable", {
                    latency: probeResult.latencyMs,
                    status: probeResult.statusCode,
                  })
                : t("providers.testUnreachable", { latency: probeResult?.latencyMs ?? 0 })}
          </p>
        ) : null}
      </CardContent>
    </Card>
  )
}
