import { Code2, Download, ExternalLink } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { AgentInstallation, AgentInstallSnapshot } from "../../services/agent-installation"
import type { AgentInstallTarget } from "./agent-installation-targets"
import type { AgentInstallations } from "./use-agent-installations"
import { AgentUpdateButton } from "./agent-update-check"

export function AgentInstallationActions({
  id,
  item,
  snapshot,
  label,
  state,
}: {
  id: AgentInstallTarget
  item?: AgentInstallation
  snapshot?: AgentInstallSnapshot
  label: string
  state: AgentInstallations
}) {
  const { t } = useTranslation()
  const pending = item && ["checking", "installing", "verifying"].includes(item.installState)
  const starting = state.install.isPending && state.install.variables === id
  return (
    <div className="flex flex-wrap gap-2">
      {id === "claude-desktop" && item?.detection.state === "installed" ? (
        <AgentUpdateButton state={state} item={item} />
      ) : null}
      {id === "claude-desktop" && item?.detection.state === "installed" ? (
        <Button size="sm" disabled={state.launch.isPending} onClick={() => state.launch.mutate()}>
          <Code2 aria-hidden="true" />
          {t("agents.install.openCode")}
        </Button>
      ) : null}
      {(id.startsWith("opencode-") ||
        id.startsWith("harness-") ||
        id.startsWith("kilocode-") ||
        id.startsWith("qwen-code-") ||
        id.startsWith("mimo-") ||
        id === "kimi-code-cli" ||
        id === "pi-cli" ||
        id === "factory-droid-cli" ||
        id.startsWith("goose-")) &&
      item?.detection.state === "installed" ? (
        <Button
          size="sm"
          disabled={state.open.isPending}
          aria-label={t("agents.install.openClient", { client: label })}
          onClick={() => state.open.mutate(id)}
        >
          <Code2 aria-hidden="true" />
          {t("agents.install.openClient", { client: label })}
        </Button>
      ) : null}
      {item?.detection.state !== "installed" && item?.canInstall !== false ? (
        <Button
          size="sm"
          disabled={
            state.busy || !snapshot?.installerAvailable || item?.detection.state !== "not_installed"
          }
          aria-label={t("agents.install.installClient", { client: label })}
          onClick={() => state.start(id)}
        >
          <Download aria-hidden="true" className={pending ? "motion-safe:animate-pulse" : ""} />
          {t(
            starting
              ? "agents.install.checking"
              : pending
                ? "agents.install.installing"
                : "agents.install.install",
          )}
        </Button>
      ) : null}
      <Button
        variant="outline"
        size="sm"
        disabled={state.official.isPending}
        onClick={() => state.official.mutate(id)}
        aria-label={t("agents.install.officialClient", { client: label })}
      >
        <ExternalLink aria-hidden="true" />
        {t("agents.install.official")}
      </Button>
    </div>
  )
}
