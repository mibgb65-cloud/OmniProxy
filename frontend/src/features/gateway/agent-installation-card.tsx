import { RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { agentInstallationError } from "../../services/agent-installation"
import type { AgentClientId } from "./agent-clients"
import { agentInstallationTargets, clientInstallationTargets } from "./agent-installation-targets"
import { agentInstallationStatus } from "./agent-installation-status"
import type { AgentInstallations } from "./use-agent-installations"
import { AgentInstallationProgress } from "./agent-installation-progress"
import { AgentUpdateStatus } from "./agent-update-check"
import { AgentInstallationActions } from "./agent-installation-actions"

export function AgentInstallationCard({
  client,
  state,
}: {
  client: AgentClientId
  state: AgentInstallations
}) {
  const { t } = useTranslation()
  const snapshot = state.data.isError ? undefined : state.data.data
  const usesInstaller = clientInstallationTargets[client].some((id) => {
    const item = snapshot?.items?.find((item) => item.id === id)
    return item && item.canInstall !== false
  })
  const installError =
    state.install.variables && clientInstallationTargets[client].includes(state.install.variables)
      ? state.install.error
      : null
  const error =
    installError ??
    (state.upgrade.variables && clientInstallationTargets[client].includes(state.upgrade.variables)
      ? state.upgrade.error
      : null) ??
    state.refresh.error ??
    state.official.error ??
    state.launch.error ??
    state.open.error
  return (
    <Card className="min-w-0 gap-4">
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-3">
        <h3 className="text-base font-semibold">{t("agents.install.title")}</h3>
        <Button
          variant="outline"
          size="sm"
          disabled={state.busy || state.update.isPending}
          onClick={state.recheck}
        >
          <RefreshCw
            aria-hidden="true"
            className={state.refresh.isPending ? "motion-safe:animate-spin" : ""}
          />
          {t("agents.install.refresh")}
        </Button>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-muted-foreground text-sm">{t("agents.install.description")}</p>
        {client === "claude-desktop" ? (
          <p className="text-muted-foreground text-sm leading-6">
            {t("agents.install.desktopCodeHint")}
          </p>
        ) : null}
        {clientInstallationTargets[client].map((id) => {
          const item = snapshot?.items?.find((item) => item.id === id)
          const label = t(agentInstallationTargets[id])
          const starting = state.install.isPending && state.install.variables === id
          const code = item?.errorCode || item?.detection.errorCode
          return (
            <div key={id} className="space-y-3 rounded-lg border p-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="min-w-0 space-y-2">
                  <h4 className="text-sm font-medium">{label}</h4>
                  <p
                    className="text-muted-foreground text-xs"
                    aria-live="polite"
                    aria-atomic="true"
                  >
                    {t(
                      starting
                        ? "agents.install.checking"
                        : snapshot
                          ? agentInstallationStatus(item)
                          : "agents.install.checking",
                    )}
                    {item?.detection.version ? ` · ${item.detection.version}` : ""}
                  </p>
                  {id === "claude-desktop" && item?.detection.state === "installed" ? (
                    <AgentUpdateStatus check={state.update} item={item} />
                  ) : null}
                </div>
                <AgentInstallationActions
                  id={id}
                  item={item}
                  snapshot={snapshot}
                  label={label}
                  state={state}
                />
              </div>
              <AgentInstallationProgress item={item} label={label} starting={starting} />
              {item?.canInstall === false && item.detection.state !== "installed" ? (
                <p className="text-muted-foreground text-xs">{t("agents.install.manual")}</p>
              ) : null}
              {code ? (
                <p role="alert" className="text-destructive text-xs">
                  {t(`agents.install.error.${agentInstallationError(`agentinstall: ${code}`)}`)}
                </p>
              ) : null}
            </div>
          )
        })}
        {state.data.isError ? (
          <p role="alert" className="text-destructive text-sm">
            {t("agents.install.loadFailed")}
          </p>
        ) : null}
        {error ? (
          <p role="alert" className="text-destructive text-sm">
            {t(`agents.install.error.${agentInstallationError(error)}`)}
          </p>
        ) : null}
        {usesInstaller && snapshot && !snapshot.installerAvailable ? (
          <p className="text-muted-foreground text-xs">
            {t("agents.install.error.installer_unavailable")}
          </p>
        ) : null}
        {usesInstaller ? (
          <p className="text-muted-foreground text-xs leading-5">
            {t("agents.install.agreements")}
          </p>
        ) : null}
      </CardContent>
    </Card>
  )
}
