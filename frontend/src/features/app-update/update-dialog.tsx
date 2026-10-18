import { X } from "lucide-react"
import { Dialog } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { formatBytes } from "../../lib/format"
import type { ApplicationUpdate } from "../../services/updates"
import { UpdateControls } from "./update-controls"

export function UpdateDialog({ state }: { state: ApplicationUpdate }) {
  const { t } = useTranslation()
  const release = state.release
  return (
    <Dialog.Portal>
      <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm" />
      <Dialog.Content className="bg-background text-foreground fixed top-1/2 left-1/2 z-50 max-h-[calc(100dvh-3rem)] w-[calc(100vw-3rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-2xl border p-6 shadow-2xl outline-none">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 space-y-1">
            <Dialog.Title className="text-lg font-semibold">{t("appUpdate.title")}</Dialog.Title>
            <Dialog.Description className="text-muted-foreground text-sm">
              {t("appUpdate.description")}
            </Dialog.Description>
          </div>
          <Dialog.Close asChild>
            <Button size="icon-sm" variant="ghost" aria-label={t("appUpdate.close")}>
              <X aria-hidden="true" />
            </Button>
          </Dialog.Close>
        </div>
        <div className="mt-5 space-y-4">
          <div className="bg-muted/50 space-y-1 rounded-xl p-3 text-sm">
            <p>{t("appUpdate.current", { version: state.currentVersion })}</p>
            <p className="font-medium">
              {t("appUpdate.target", { version: release?.version ?? state.download.version })}
            </p>
            {release?.publishedAt ? (
              <p className="text-muted-foreground">
                {new Date(release.publishedAt).toLocaleString()}
              </p>
            ) : null}
            {release?.size ? (
              <p className="text-muted-foreground">{formatBytes(release.size)}</p>
            ) : null}
          </div>
          <section className="space-y-2">
            <h3 className="text-sm font-medium">{t("appUpdate.notes")}</h3>
            <p className="text-muted-foreground max-h-52 overflow-y-auto text-sm break-words whitespace-pre-wrap">
              {release?.notes || t("appUpdate.noNotes")}
            </p>
          </section>
          <UpdateControls state={state} />
          {release?.url ? (
            <a
              href={release.url}
              target="_blank"
              rel="noreferrer"
              className="text-primary inline-block text-sm underline underline-offset-4"
            >
              {t("about.update.openRelease")}
            </a>
          ) : null}
        </div>
      </Dialog.Content>
    </Dialog.Portal>
  )
}
