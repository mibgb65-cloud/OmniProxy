import { Copy, Power, RefreshCw, RotateCcw, Settings2 } from "lucide-react"
import { useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { agentError, rereadAgentConfig, type AgentClient } from "../../services/agents"
import { AgentConfigDialog } from "./agent-config-dialog"
import { useAgentActions } from "./use-agent-actions"

export function AgentClientSettings({
  item,
  models,
  running,
}: {
  item: AgentClient
  models: string[]
  running: boolean
}) {
  const { t } = useTranslation()
  const actions = useAgentActions(item)
  // 预览与应用使用打开时的文件版本，后台刷新不能替换该快照。
  const [selected, setSelected] = useState<AgentClient | null>(null)
  const queryClient = useQueryClient()
  const [reread, setReread] = useState(false)
  const [reloading, setReloading] = useState(false)
  const [reloadError, setReloadError] = useState<ReturnType<typeof agentError> | null>(null)
  async function reload() {
    setReloading(true)
    setReloadError(null)
    try {
      const next = await rereadAgentConfig(item.client)
      await queryClient.invalidateQueries({ queryKey: ["agent-clients"] })
      setReread(true)
      setSelected(next)
    } catch (error) {
      setReloadError(agentError(error))
    } finally {
      setReloading(false)
    }
  }
  return (
    <>
      <Card className="min-w-0 gap-5">
        <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-3">
          <h3 className="text-base font-semibold">{t("agents.connectionSettings")}</h3>
          <Button
            variant="outline"
            size="sm"
            disabled={actions.busy || reloading}
            onClick={() => void reload()}
          >
            <RefreshCw aria-hidden="true" className={reloading ? "motion-safe:animate-spin" : ""} />
            {t("agents.reread")}
          </Button>
        </CardHeader>
        <CardContent className="space-y-5">
          <dl className="grid min-w-0 gap-5 text-sm sm:grid-cols-2">
            <div className="min-w-0 space-y-2 sm:col-span-2">
              <dt className="text-muted-foreground">{t("agents.configPath")}</dt>
              <dd className="bg-muted/30 rounded-lg border px-3 py-2.5 font-mono text-xs leading-6 wrap-anywhere">
                {item.path}
              </dd>
            </div>
            <div className="space-y-2">
              <dt className="text-muted-foreground">{t("agents.configFile")}</dt>
              <dd>{t(item.exists ? "agents.fileExists" : "agents.fileMissing")}</dd>
            </div>
            {item.provider ? (
              <div className="min-w-0 space-y-2">
                <dt className="text-muted-foreground">{t("agents.provider")}</dt>
                <dd className="font-mono wrap-anywhere">{item.provider}</dd>
              </div>
            ) : null}
          </dl>
          <div className="flex flex-wrap gap-2 border-t pt-5">
            <Button
              disabled={actions.busy || !models.length || !!item.errorCode}
              onClick={() => {
                setReread(false)
                setSelected(item)
              }}
            >
              <Settings2 aria-hidden="true" />
              {t("agents.configure")}
            </Button>
            {item.canRestore ? (
              <>
                <Button
                  variant="outline"
                  disabled={actions.busy}
                  onClick={() => actions.run("copy")}
                >
                  <Copy aria-hidden="true" />
                  {t("agents.copyKey")}
                </Button>
                <Button
                  variant="outline"
                  disabled={actions.busy}
                  onClick={() => actions.run("toggle")}
                >
                  <Power aria-hidden="true" />
                  {t(item.configured ? "agents.disable" : "agents.enable")}
                </Button>
                <Button
                  variant="outline"
                  disabled={actions.busy || !!item.errorCode}
                  onClick={() => actions.run("restore")}
                >
                  <RotateCcw aria-hidden="true" />
                  {t("agents.restore")}
                </Button>
              </>
            ) : null}
          </div>
          {item.errorCode || actions.error || reloadError ? (
            <p role="alert" className="text-destructive text-sm">
              {t(
                `agents.error.${reloadError ?? actions.error ?? agentError(`agent: ${item.errorCode}`)}`,
              )}
            </p>
          ) : null}
          {actions.busy ? (
            <p role="status" className="text-muted-foreground text-sm">
              {t("agents.working")}
            </p>
          ) : null}
          {actions.notice ? (
            <p role="status" className="text-sm text-emerald-700 dark:text-emerald-400">
              {t(actions.notice)}
            </p>
          ) : null}
          {!models.length ? (
            <p className="text-muted-foreground text-sm">{t("agents.noModels")}</p>
          ) : null}
          {!running ? <p className="text-muted-foreground text-sm">{t("agents.stopped")}</p> : null}
        </CardContent>
      </Card>
      <AgentConfigDialog item={selected} reread={reread} onClose={() => setSelected(null)} />
    </>
  )
}
