import { X } from "lucide-react"
import { Dialog } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Markdown } from "../../components/common/markdown"
import { Button } from "../../components/ui/button"
import type { ApplicationUpdate } from "../../services/updates"
import { UpdateControls } from "./update-controls"
import { UpdateVersionSummary } from "./update-version-summary"

/**
 * 升级浮窗：标题栏与操作区固定，中间是可滚动的版本摘要与发布说明，
 * 说明正文按 Markdown 排版而不是原样输出源码。
 */
export function UpdateDialog({ state }: { state: ApplicationUpdate }) {
  const { t } = useTranslation()
  const notes = state.release?.notes.trim()
  return (
    <Dialog.Portal>
      <Dialog.Overlay className="motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 fixed inset-0 z-40 bg-black/50 backdrop-blur-sm" />
      <Dialog.Content className="bg-card text-card-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 motion-safe:data-[state=closed]:zoom-out-95 fixed top-1/2 left-1/2 z-50 flex max-h-[calc(100dvh-2rem)] w-[min(720px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-2xl border shadow-2xl outline-none">
        <header className="flex shrink-0 items-start justify-between gap-4 border-b px-6 py-4">
          <div className="min-w-0 space-y-1">
            <Dialog.Title className="text-lg font-semibold tracking-tight">
              {t("appUpdate.title")}
            </Dialog.Title>
            <Dialog.Description className="text-muted-foreground text-sm">
              {t("appUpdate.description")}
            </Dialog.Description>
          </div>
          <Dialog.Close asChild>
            <Button size="icon-sm" variant="ghost" aria-label={t("appUpdate.close")}>
              <X aria-hidden="true" />
            </Button>
          </Dialog.Close>
        </header>
        <div className="app-scroll min-h-0 flex-1 space-y-5 overflow-y-auto px-6 py-5">
          <UpdateVersionSummary state={state} />
          <section className="space-y-2">
            <h3 className="text-sm font-medium">{t("appUpdate.notes")}</h3>
            <div className="bg-muted/20 rounded-xl border px-4 py-3">
              {notes ? (
                <Markdown source={notes} />
              ) : (
                <p className="text-muted-foreground text-sm">{t("appUpdate.noNotes")}</p>
              )}
            </div>
          </section>
        </div>
        <footer className="shrink-0 border-t px-6 py-4">
          <UpdateControls state={state} />
        </footer>
      </Dialog.Content>
    </Dialog.Portal>
  )
}
