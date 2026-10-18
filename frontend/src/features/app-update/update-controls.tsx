import { Download, LoaderCircle, RotateCw, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { formatBytes } from "../../lib/format"
import type { ApplicationUpdate } from "../../services/updates"
import { useApplicationUpdateActions } from "./use-application-update"

export function UpdateControls({ state }: { state: ApplicationUpdate }) {
  const { t } = useTranslation()
  const { download, cancel, install } = useApplicationUpdateActions()
  const transfer = state.download
  const downloading = transfer.state === "downloading"
  const installing = ["verifying", "installing"].includes(transfer.state)
  const ready = transfer.state === "ready"
  const busy =
    downloading || installing || download.isPending || install.isPending || cancel.isPending
  const error = download.error ?? cancel.error ?? install.error
  const target = state.release?.version
  return (
    <div className="space-y-3">
      {downloading ? (
        <div className="space-y-2" aria-live="polite">
          <div className="flex flex-wrap justify-between gap-2 text-sm">
            <span>{t("appUpdate.downloading")}</span>
            <span>
              {transfer.percent}% · {formatBytes(transfer.received)} / {formatBytes(transfer.total)}
            </span>
          </div>
          <div
            role="progressbar"
            aria-label={t("appUpdate.progress")}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={transfer.percent}
            className="bg-muted h-2 overflow-hidden rounded-full"
          >
            <div
              className="bg-primary h-full transition-[width] motion-reduce:transition-none"
              style={{ width: `${Math.max(0, Math.min(100, transfer.percent))}%` }}
            />
          </div>
        </div>
      ) : null}
      {ready ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("appUpdate.ready", { version: transfer.version })}
        </p>
      ) : null}
      {installing ? (
        <p role="status" className="flex items-center gap-2 text-sm">
          <LoaderCircle
            className="size-4 animate-spin motion-reduce:animate-none"
            aria-hidden="true"
          />
          {t(transfer.state === "verifying" ? "appUpdate.verifying" : "appUpdate.installing")}
        </p>
      ) : null}
      {transfer.state === "cancelled" ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("appUpdate.cancelled")}
        </p>
      ) : null}
      {transfer.error || error ? (
        <p role="alert" className="text-destructive text-sm">
          {String(transfer.error || error?.message).includes("checksum")
            ? t("appUpdate.checksumFailed")
            : t("appUpdate.failed")}
        </p>
      ) : null}
      {!state.canInstall && state.release ? (
        <p className="text-muted-foreground text-sm">{t("appUpdate.installerRequired")}</p>
      ) : null}
      {state.release && !state.release.packageAvailable ? (
        <p className="text-muted-foreground text-sm">{t("appUpdate.noPackage")}</p>
      ) : null}
      <div className="flex flex-wrap gap-2">
        {downloading ? (
          <Button
            size="sm"
            variant="outline"
            disabled={cancel.isPending}
            onClick={() => cancel.mutate()}
          >
            <X aria-hidden="true" />
            {t("appUpdate.cancel")}
          </Button>
        ) : null}
        {state.canInstall && ready ? (
          <Button size="sm" disabled={busy} onClick={() => install.mutate(transfer.version)}>
            <RotateCw aria-hidden="true" />
            {t("appUpdate.restart")}
          </Button>
        ) : null}
        {state.canInstall &&
        state.release?.packageAvailable &&
        !ready &&
        !downloading &&
        !installing ? (
          <Button
            size="sm"
            disabled={busy}
            onClick={() => {
              if (target) download.mutate(target)
            }}
          >
            <Download aria-hidden="true" />
            {download.isPending ? t("appUpdate.starting") : t("appUpdate.download")}
          </Button>
        ) : null}
      </div>
      {ready && state.canInstall ? (
        <p className="text-muted-foreground text-xs">{t("appUpdate.restartHint")}</p>
      ) : null}
    </div>
  )
}
