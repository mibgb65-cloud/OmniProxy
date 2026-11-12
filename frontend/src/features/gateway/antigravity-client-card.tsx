import { useQueryClient } from "@tanstack/react-query"
import { AlertTriangle, Link2, Loader2, RotateCcw } from "lucide-react"
import { useId, useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import {
  agentError,
  applyAgentConfig,
  previewAgentPlan,
  restoreAgentConfig,
  type AgentClient,
  type AgentPlan,
} from "../../services/agents"

// Antigravity 桌面端通过官方 Enterprise Gateway 读取本地网关；不需要改代理或证书。
// 版式与 AgentClientGroupTile 一致：它没有安装目标和详情页，所以不链接到详情，操作按钮直接放在卡片底部。
export function AntigravityGroupTile({ item, pending }: { item?: AgentClient; pending: boolean }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const titleId = useId()
  const [plan, setPlan] = useState<AgentPlan | null>(null)
  const [model, setModel] = useState("")
  const [error, setError] = useState<ReturnType<typeof agentError> | null>(null)
  const [busy, setBusy] = useState(false)
  const [saved, setSaved] = useState(false)
  const configured = !!item?.configured && !item.errorCode
  const missing = !!item && !item.exists && !item.configured
  const installed = !!item?.exists
  // Antigravity 的「已配置」是写入官方网关设置，不是接入 Key，用专门的状态文案。
  const statusKey = !item
    ? "agents.unavailable"
    : item.errorCode
      ? "agents.needsAttention"
      : item.configured
        ? "agents.antigravityConnected"
        : "agents.idle"
  const installState = !item
    ? pending
      ? "agents.install.checking"
      : "agents.install.unknown"
    : installed
      ? "agents.install.installed"
      : "agents.install.notInstalled"

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
    <section
      aria-labelledby={titleId}
      className="bg-card flex min-w-0 flex-col gap-5 rounded-xl border p-6"
    >
      <header className="flex items-center gap-4">
        <span
          aria-hidden="true"
          className="bg-muted/30 flex size-14 shrink-0 items-center justify-center rounded-2xl border"
        >
          <img
            src="/provider-icons/google-color.svg"
            alt=""
            width={32}
            height={32}
            className="size-8"
          />
        </span>
        <div className="min-w-0 space-y-2">
          <h3 id={titleId} className="text-base font-semibold">
            {t("agents.antigravityGroup")}
          </h3>
          <p className="text-muted-foreground text-sm leading-6">
            {t("agents.antigravityDescription")}
          </p>
        </div>
      </header>
      <div className="grid gap-3">
        <div className="bg-card flex min-w-0 flex-col gap-2 rounded-xl border p-3">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <h4 className="text-sm font-medium">{t("agents.antigravityTitle")}</h4>
            <span
              className={`flex items-center gap-1.5 rounded-full border px-2 py-1 text-xs ${configured ? "border-emerald-500/20 text-emerald-700 dark:text-emerald-400" : "text-muted-foreground"}`}
            >
              <span
                aria-hidden="true"
                className={`size-1.5 shrink-0 rounded-full ${configured ? "bg-emerald-500" : "bg-muted-foreground/60"}`}
              />
              {t(pending ? "agents.loading" : statusKey)}
            </span>
          </div>
          <dl className="space-y-1.5 text-xs">
            <dt className="sr-only">{t("agents.antigravityTitle")}</dt>
            <dd
              className={
                installed ? "text-emerald-700 dark:text-emerald-400" : "text-muted-foreground"
              }
            >
              {t(installState)}
            </dd>
          </dl>
          {missing ? (
            <p role="status" className="text-muted-foreground text-xs leading-5">
              {t("agents.antigravityNoConfig")}
            </p>
          ) : null}
          {plan ? (
            <label className="block space-y-2 text-sm">
              <span className="text-muted-foreground text-xs">
                {t("agents.antigravityDefaultModel")}
              </span>
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
          ) : null}
          <p className="text-muted-foreground text-xs leading-5">{t("agents.antigravityHint")}</p>
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
        </div>
      </div>
      <footer className="mt-auto space-y-2 border-t pt-4">
        {plan ? (
          <div className="flex gap-2">
            <Button
              size="sm"
              className="flex-1"
              disabled={busy || !model}
              onClick={() => void apply()}
            >
              {busy ? (
                <Loader2 aria-hidden="true" className="motion-safe:animate-spin" />
              ) : (
                <Link2 aria-hidden="true" />
              )}
              {t("agents.antigravityApply")}
            </Button>
            <Button variant="outline" size="sm" disabled={busy} onClick={() => setPlan(null)}>
              {t("agents.close")}
            </Button>
          </div>
        ) : configured ? (
          <Button
            variant="outline"
            size="sm"
            className="w-full"
            disabled={busy}
            onClick={() => void disconnect()}
          >
            {busy ? (
              <Loader2 aria-hidden="true" className="motion-safe:animate-spin" />
            ) : (
              <RotateCcw aria-hidden="true" />
            )}
            {t("agents.antigravityDisconnect")}
          </Button>
        ) : (
          <Button
            variant="outline"
            size="sm"
            className="w-full"
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
        )}
        {saved ? (
          <p role="status" className="text-xs leading-5 text-emerald-700 dark:text-emerald-400">
            {t("agents.success")} {t("agents.antigravityRestart")}
          </p>
        ) : null}
        {configured ? (
          <p className="text-muted-foreground text-xs leading-5">
            {t("agents.antigravityGoogleOff")}
          </p>
        ) : null}
      </footer>
    </section>
  )
}
