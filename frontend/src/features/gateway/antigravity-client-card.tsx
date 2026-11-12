import { useQueryClient } from "@tanstack/react-query"
import { AlertTriangle, Link2, Loader2, RotateCcw } from "lucide-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import {
  agentError,
  applyAgentConfig,
  previewAgentPlan,
  restoreAgentConfig,
  type AgentClient,
  type AgentPlan,
} from "../../services/agents"
import { agentStatusKey } from "./agent-clients"

// Antigravity 桌面端通过官方 Enterprise Gateway 读取本地网关；不需要改代理或证书。
export function AntigravityClientCard({ item, pending }: { item?: AgentClient; pending: boolean }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [plan, setPlan] = useState<AgentPlan | null>(null)
  const [model, setModel] = useState("")
  const [error, setError] = useState<ReturnType<typeof agentError> | null>(null)
  const [busy, setBusy] = useState(false)
  const [saved, setSaved] = useState(false)
  const configured = !!item?.configured && !item.errorCode
  const missing = !!item && !item.exists && !item.configured

  async function refresh() {
    await queryClient.invalidateQueries({ queryKey: ["agent-clients"] })
    await queryClient.invalidateQueries({ queryKey: ["agent-configured-models"] })
  }

  async function connect() {
    if (!item) return
    setBusy(true)
    setError(null)
    setSaved(false)
    try {
      const next = await previewAgentPlan({ client: "antigravity", revision: item.revision })
      setPlan(next)
      setModel(next.defaultModel || (next.models ?? [])[0]?.id || "")
    } catch (cause) {
      setError(agentError(cause))
    } finally {
      setBusy(false)
    }
  }

  async function apply() {
    if (!item || !plan) return
    setBusy(true)
    setError(null)
    try {
      await applyAgentConfig({
        client: "antigravity",
        revision: item.revision,
        catalogRevision: plan.catalogRevision,
        model,
      })
      setPlan(null)
      setSaved(true)
      await refresh()
    } catch (cause) {
      setError(agentError(cause))
    } finally {
      setBusy(false)
    }
  }

  async function disconnect() {
    setBusy(true)
    setError(null)
    setSaved(false)
    try {
      await restoreAgentConfig("antigravity")
      setPlan(null)
      await refresh()
    } catch (cause) {
      setError(agentError(cause))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Card className="min-w-0 gap-4">
      <CardHeader className="flex flex-row flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-3">
          <span
            aria-hidden="true"
            className="bg-muted/30 flex size-11 shrink-0 items-center justify-center rounded-xl border"
          >
            <img src="/provider-icons/google-color.svg" alt="" width={26} height={26} />
          </span>
          <div className="min-w-0 space-y-1">
            <h2 className="text-base font-semibold">{t("agents.antigravityTitle")}</h2>
            <p className="text-muted-foreground text-sm">{t("agents.antigravityDescription")}</p>
          </div>
        </div>
        <span className="text-muted-foreground rounded-full border px-2.5 py-1 text-xs">
          {t(pending ? "agents.loading" : agentStatusKey(item))}
        </span>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-muted-foreground text-sm leading-6">{t("agents.antigravityHint")}</p>
        {configured ? (
          <div className="space-y-3">
            <p className="text-sm font-medium text-emerald-700 dark:text-emerald-400">
              {t("agents.antigravityConnected")}
            </p>
            <p className="text-muted-foreground text-sm leading-6">
              {t("agents.antigravityRestart")} {t("agents.antigravityGoogleOff")}
            </p>
            <Button variant="outline" disabled={busy} onClick={() => void disconnect()}>
              {busy ? (
                <Loader2 aria-hidden="true" className="motion-safe:animate-spin" />
              ) : (
                <RotateCcw aria-hidden="true" />
              )}
              {t("agents.antigravityDisconnect")}
            </Button>
          </div>
        ) : plan ? (
          <div className="space-y-3">
            <label className="block space-y-2 text-sm">
              <span className="text-muted-foreground">{t("agents.antigravityDefaultModel")}</span>
              <select
                className="bg-background focus-visible:ring-ring h-9 w-full rounded-md border px-3 text-sm outline-none focus-visible:ring-2"
                value={model}
                onChange={(event) => setModel(event.target.value)}
              >
                {(plan.models ?? []).map((entry) => (
                  <option key={entry.id} value={entry.id}>
                    {entry.name || entry.id}
                  </option>
                ))}
              </select>
            </label>
            <div className="flex flex-wrap gap-2">
              <Button disabled={busy || !model} onClick={() => void apply()}>
                {busy ? (
                  <Loader2 aria-hidden="true" className="motion-safe:animate-spin" />
                ) : (
                  <Link2 aria-hidden="true" />
                )}
                {t("agents.antigravityApply")}
              </Button>
              <Button variant="outline" disabled={busy} onClick={() => setPlan(null)}>
                {t("agents.close")}
              </Button>
            </div>
            <p className="text-muted-foreground text-xs leading-5 whitespace-pre-line">
              {plan.summary}
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            {missing ? (
              <p role="status" className="text-muted-foreground text-sm leading-6">
                {t("agents.antigravityNoConfig")}
              </p>
            ) : null}
            <Button
              disabled={pending || busy || !item || missing || !!item?.errorCode}
              onClick={() => void connect()}
            >
              {busy ? (
                <Loader2 aria-hidden="true" className="motion-safe:animate-spin" />
              ) : (
                <Link2 aria-hidden="true" />
              )}
              {t("agents.antigravityConnect")}
            </Button>
          </div>
        )}
        {item?.path ? (
          <p className="text-muted-foreground flex min-w-0 gap-1 text-xs leading-5">
            <span className="shrink-0">{t("agents.antigravityPath")}:</span>
            <code className="min-w-0 wrap-anywhere">{item.path}</code>
          </p>
        ) : null}
        {error ? (
          <p role="alert" className="text-destructive flex items-start gap-2 text-sm">
            <AlertTriangle aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
            {t(`agents.error.${error}`, { defaultValue: t("agents.error.failed") })}
          </p>
        ) : null}
        {saved ? (
          <p role="status" className="text-sm text-emerald-700 dark:text-emerald-400">
            {t("agents.success")} {t("agents.antigravityRestart")}
          </p>
        ) : null}
      </CardContent>
    </Card>
  )
}
