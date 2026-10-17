import { Play, RefreshCw, Square } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { gatewayErrorKey, gatewayOperationErrorKey } from "./gateway-errors"
import type { GatewayPreparation } from "./use-gateway-preparation"
import { GatewayActionIcon } from "./gateway-action-icon"

export function GatewayHeader({ data }: { data: GatewayPreparation }) {
  const { t } = useTranslation()
  const { gateway } = data
  const stateLabel = t(gateway.statusKey)
  const runtimeError = gateway.state === "error" && gateway.info.data?.runtime.errorCode
  const toggleLabel = gateway.controlsBusy
    ? stateLabel
    : t(gateway.isRunning ? "gateway.status.stop" : "gateway.status.start")

  return (
    <header className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="min-w-0 space-y-2">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="text-2xl font-semibold tracking-tight">{t("pages.gateway.title")}</h1>
            <span
              aria-live="polite"
              aria-atomic="true"
              className="bg-muted text-muted-foreground rounded-full border px-2.5 py-1 text-xs font-medium"
            >
              <span className="sr-only">{t("gateway.status.title")}：</span>
              {stateLabel}
            </span>
          </div>
          <code className="text-muted-foreground block text-sm wrap-anywhere">
            {data.address?.replace(/\/v1$/, "") ??
              t(data.runtimeState === "loading" ? "gateway.loading" : "gateway.unavailable")}
          </code>
        </div>
        <div className="space-y-2">
          <div className="flex flex-wrap gap-2">
            <Button
              className="gateway-action-button px-3"
              disabled={gateway.controlsBusy || !gateway.info.isSuccess}
              aria-busy={gateway.controlsBusy}
              onClick={gateway.toggle}
              aria-describedby="gateway-control-notice"
            >
              <GatewayActionIcon
                pending={gateway.controlsBusy}
                icon={gateway.isRunning ? Square : Play}
              />
              <StableActionLabel
                label={toggleLabel}
                labels={[
                  t("gateway.status.start"),
                  t("gateway.status.stop"),
                  t("gateway.status.starting"),
                  t("gateway.status.stopping"),
                ]}
              />
            </Button>
            <Button
              variant="outline"
              className="gateway-action-button"
              disabled={data.isRefreshing || data.checksLoading}
              aria-busy={data.isRefreshing}
              onClick={() => void data.refresh()}
            >
              <RefreshCw
                aria-hidden="true"
                className={data.isRefreshing ? "motion-safe:animate-spin" : ""}
              />
              <StableActionLabel
                label={t(data.isRefreshing ? "gateway.refreshing" : "gateway.refresh")}
                labels={[t("gateway.refresh"), t("gateway.refreshing")]}
              />
            </Button>
          </div>
          <p id="gateway-control-notice" className="text-muted-foreground text-xs leading-5">
            {t("gateway.status.note")}
          </p>
        </div>
      </div>
      {gateway.info.isError ? (
        <p className="text-destructive text-sm" role="alert">
          {t("gateway.status.loadFailed")}
        </p>
      ) : null}
      {gateway.operationError ? (
        <p className="text-destructive text-sm" role="alert">
          {t(gatewayOperationErrorKey(gateway.operationError))}
        </p>
      ) : runtimeError ? (
        <p className="text-destructive text-sm" role="alert">
          {t(gatewayErrorKey(runtimeError))}
        </p>
      ) : null}
      {data.runtimeState === "error" ? (
        <p className="text-destructive text-sm" role="alert">
          {t(
            data.runtime.isError
              ? "gateway.connection.loadFailed"
              : "gateway.connection.invalidPort",
          )}
        </p>
      ) : null}
    </header>
  )
}

function StableActionLabel({ label, labels }: { label: string; labels: string[] }) {
  return (
    <span className="inline-grid">
      {/* 所有状态共用网格单元预留最大文案宽度，避免启停／刷新时挤动按钮；隐藏文案不参与播报。 */}
      {labels.map((text) => (
        <span key={text} aria-hidden="true" className="invisible col-start-1 row-start-1">
          {text}
        </span>
      ))}
      <span className="col-start-1 row-start-1">{label}</span>
    </span>
  )
}
