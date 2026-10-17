import { RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { AgentInstallations } from "./use-agent-installations"

type Check = AgentInstallations["update"]

export function AgentUpdateButton({ check, disabled }: { check: Check; disabled: boolean }) {
  const { t } = useTranslation()
  return (
    <Button
      variant="outline"
      size="sm"
      className="transition-colors"
      disabled={disabled || check.isPending}
      aria-busy={check.isPending}
      onClick={() => check.mutate("claude-desktop")}
    >
      <RefreshCw
        aria-hidden="true"
        className={check.isPending ? "motion-safe:animate-[spin_1.4s_linear_infinite]" : ""}
      />
      <span className="inline-grid">
        <span aria-hidden="true" className="invisible col-start-1 row-start-1">
          {t("agents.update.check")}
        </span>
        <span aria-hidden="true" className="invisible col-start-1 row-start-1">
          {t("agents.update.checking")}
        </span>
        <span className="col-start-1 row-start-1">
          {t(check.isPending ? "agents.update.checking" : "agents.update.check")}
        </span>
      </span>
    </Button>
  )
}

export function AgentUpdateStatus({ check }: { check: Check }) {
  const { t } = useTranslation()
  if (check.variables !== "claude-desktop" || check.isPending) return null
  const result = check.data
  const message = check.isError
    ? t("agents.update.failed")
    : result?.state === "available"
      ? t("agents.update.available", { version: result.latestVersion })
      : result?.state === "current"
        ? t("agents.update.current")
        : result
          ? t("agents.update.unknown")
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
