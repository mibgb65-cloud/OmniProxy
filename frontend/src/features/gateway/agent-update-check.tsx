import { Download, RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { AgentInstallations } from "./use-agent-installations"
import type { AgentInstallation } from "../../services/agent-installation"

type Check = AgentInstallations["update"]

export function AgentUpdateButton({
  state,
  item,
}: {
  state: AgentInstallations
  item?: AgentInstallation
}) {
  const { t } = useTranslation()
  const check = state.update
  const updating =
    state.upgrade.isPending ||
    (item?.operation === "update" &&
      ["checking", "installing", "verifying"].includes(item.installState))
  const available = check.variables === "claude-desktop" && check.data?.state === "available"
  const pending = check.isPending || updating
  const labels = [
    "agents.update.check",
    "agents.update.checking",
    "agents.update.install",
    "agents.update.updating",
  ] as const
  const label = updating
    ? labels[3]
    : check.isPending
      ? labels[1]
      : available
        ? labels[2]
        : labels[0]
  const Icon = available && !pending ? Download : RefreshCw
  return (
    <Button
      variant="outline"
      size="sm"
      className="transition-colors"
      disabled={state.busy || pending || (available && !state.data.data?.installerAvailable)}
      aria-busy={pending}
      onClick={() =>
        available ? state.startUpdate("claude-desktop") : check.mutate("claude-desktop")
      }
    >
      <Icon
        aria-hidden="true"
        className={pending ? "motion-safe:animate-[spin_1.4s_linear_infinite]" : ""}
      />
      <span className="inline-grid">
        {labels.map((key) => (
          <span key={key} aria-hidden="true" className="invisible col-start-1 row-start-1">
            {t(key)}
          </span>
        ))}
        <span className="col-start-1 row-start-1">{t(label)}</span>
      </span>
    </Button>
  )
}

export function AgentUpdateStatus({ check, item }: { check: Check; item?: AgentInstallation }) {
  const { t } = useTranslation()
  if (
    check.isPending ||
    (item?.operation === "update" &&
      ["checking", "installing", "verifying"].includes(item.installState))
  )
    return null
  const result = check.data
  const message = check.isError
    ? t("agents.update.failed")
    : result?.state === "available"
      ? t("agents.update.available", { version: result.latestVersion })
      : result?.state === "current"
        ? t("agents.update.current")
        : result
          ? t("agents.update.unknown")
          : item?.operation === "update" && item.installState === "completed"
            ? t("agents.update.completed", { version: item.detection.version })
            : null
  return message ? (
    <p
      role="status"
      className={check.isError ? "text-destructive text-xs" : "text-muted-foreground text-xs"}
    >
      {message}
    </p>
  ) : null
}
