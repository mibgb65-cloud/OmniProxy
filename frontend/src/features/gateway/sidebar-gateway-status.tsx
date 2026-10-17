import { Power } from "lucide-react"
import { useId } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import { cn } from "../../lib/utils"
import { useSidebarGatewayStatus } from "./use-sidebar-gateway-status"
import "./sidebar-gateway-status.css"
import { GatewayActionIcon } from "./gateway-action-icon"

export function SidebarGatewayStatus({ collapsed }: { collapsed: boolean }) {
  const { t } = useTranslation()
  const { gateway, port, accountCount, accountsLoading, errorKey } = useSidebarGatewayStatus()
  const statusId = useId()
  const errorId = useId()
  const stateLabel = t(gateway.statusKey)
  const actionLabel = gateway.controlsBusy
    ? stateLabel
    : t(gateway.isRunning ? "gateway.status.stop" : "gateway.status.start")
  const portLabel = port ?? t(gateway.info.isPending ? "gateway.loading" : "gateway.unavailable")
  const accountsLabel =
    accountCount ?? t(accountsLoading ? "gateway.loading" : "gateway.unavailable")
  const statusColor = gateway.controlsBusy
    ? "bg-amber-500"
    : gateway.state === "running"
      ? "bg-emerald-500"
      : gateway.state === "error" || gateway.info.isError
        ? "bg-destructive"
        : "bg-muted-foreground/50"
  const tooltip = collapsed ? (
    <div className="space-y-1">
      <p>
        {t("gateway.sidebar.title")} · {stateLabel}
      </p>
      <p>
        {t("gateway.sidebar.port")}：{portLabel} · {t("gateway.sidebar.accounts")}：{accountsLabel}
      </p>
      <p>{actionLabel}</p>
      {gateway.isRunning ? <p>{t("gateway.status.note")}</p> : null}
      {errorKey ? <p>{t(errorKey)}</p> : null}
    </div>
  ) : gateway.isRunning ? (
    <div>
      {actionLabel}
      <br />
      {t("gateway.status.note")}
    </div>
  ) : (
    actionLabel
  )

  return (
    <section
      aria-label={t("gateway.sidebar.title")}
      data-collapsed={collapsed}
      className="sidebar-gateway border-sidebar-border shrink-0 border-b p-2"
    >
      {/* 状态播报独立于视觉裁切，收起时仍可读取按钮描述和错误。 */}
      <p id={statusId} role="status" aria-atomic="true" className="sr-only">
        {t("gateway.sidebar.title")}：{stateLabel}
      </p>
      {errorKey ? (
        <p id={errorId} role="alert" className="sr-only">
          {t(errorKey)}
        </p>
      ) : null}
      <div className="sidebar-gateway-card">
        <div
          className="sidebar-gateway-surface bg-background/70 border-sidebar-border rounded-xl border shadow-xs"
          aria-hidden="true"
        />
        <div className="sidebar-gateway-heading" aria-hidden={collapsed}>
          <div className="h-10 pr-10 whitespace-nowrap">
            <p className="text-muted-foreground text-xs">{t("gateway.sidebar.title")}</p>
            <p className="mt-1 flex items-center gap-2 text-sm font-medium">
              <span
                className={cn("size-2 shrink-0 rounded-full", statusColor)}
                aria-hidden="true"
              />
              {stateLabel}
            </p>
          </div>
        </div>
        <div className="sidebar-gateway-body" aria-hidden={collapsed}>
          <div className="sidebar-gateway-viewport">
            <div className="sidebar-gateway-details">
              <dl className="border-sidebar-border grid grid-cols-2 gap-3 border-t pt-2">
                <div className="min-w-0">
                  <dt className="text-muted-foreground text-xs">{t("gateway.sidebar.port")}</dt>
                  <dd className="mt-1 text-sm font-medium tabular-nums">{portLabel}</dd>
                </div>
                <div className="min-w-0" title={t("gateway.sidebar.accountsHint")}>
                  <dt className="text-muted-foreground text-xs">{t("gateway.sidebar.accounts")}</dt>
                  <dd className="mt-1 text-sm font-medium tabular-nums">{accountsLabel}</dd>
                </div>
              </dl>
              {errorKey ? (
                <p aria-hidden="true" className="text-destructive mt-2 text-xs leading-5">
                  {t(errorKey)}
                </p>
              ) : null}
            </div>
          </div>
        </div>
        <Tooltip content={tooltip} side="right">
          <span
            className="sidebar-gateway-power inline-flex"
            tabIndex={gateway.controlsBusy || !gateway.info.isSuccess ? 0 : undefined}
          >
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label={actionLabel}
              aria-describedby={errorKey ? `${statusId} ${errorId}` : statusId}
              aria-busy={gateway.controlsBusy}
              disabled={gateway.controlsBusy || !gateway.info.isSuccess}
              onClick={gateway.toggle}
              className={cn(
                "gateway-action-button cursor-pointer rounded-lg border",
                gateway.isRunning && !gateway.controlsBusy
                  ? "border-emerald-200 bg-emerald-50 text-emerald-700 hover:bg-emerald-100 dark:border-emerald-800 dark:bg-emerald-950 dark:text-emerald-300 dark:hover:bg-emerald-900"
                  : "border-border bg-muted/40 text-muted-foreground hover:bg-muted hover:text-foreground",
              )}
            >
              <GatewayActionIcon pending={gateway.controlsBusy} icon={Power} />
            </Button>
          </span>
        </Tooltip>
        <span
          className={cn("sidebar-gateway-dot size-1.5 rounded-full", statusColor)}
          aria-hidden="true"
        />
      </div>
    </section>
  )
}
