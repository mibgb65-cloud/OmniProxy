import { ArrowRight, PartyPopper, ScrollText, X } from "lucide-react"
import { Dialog } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Markdown } from "../../components/common/markdown"
import { Button } from "../../components/ui/button"
import type { ApplicationUpdateInstalled } from "../../services/updates"
import { getCurrentReleaseNotes, withoutTopHeading } from "../about/current-release"

/**
 * 更新完成浮窗：升级并重启之后的一次性反馈，沿用升级浮窗的三段式结构
 * （固定标题栏 + 可滚动正文 + 固定操作区）。正文只取本地内置的当前版本发布说明，
 * 不为了这条提示额外联网。
 */
export function UpdateCompletedDialog({
  installed,
  onConfirm,
  onOpenNotes,
}: {
  installed: ApplicationUpdateInstalled
  onConfirm: () => void
  onOpenNotes: () => void
}) {
  const { t } = useTranslation()
  const notes = getCurrentReleaseNotes(installed.version)?.trim()
  return (
    <Dialog.Portal>
      <Dialog.Overlay className="motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 fixed inset-0 z-40 bg-black/50 backdrop-blur-sm motion-safe:duration-150" />
      <Dialog.Content className="bg-card text-card-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 motion-safe:data-[state=closed]:zoom-out-95 fixed top-1/2 left-1/2 z-50 flex max-h-[calc(100dvh-2rem)] w-[min(680px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-2xl border shadow-2xl outline-none motion-safe:duration-150">
        <header className="flex shrink-0 items-start justify-between gap-4 border-b px-6 py-4">
          <div className="flex min-w-0 gap-3">
            <div className="bg-primary/10 text-primary mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-xl">
              <PartyPopper className="size-4.5" aria-hidden="true" />
            </div>
            <div className="min-w-0 space-y-1">
              <Dialog.Title className="text-lg font-semibold tracking-tight">
                {t("appUpdate.completedTitle")}
              </Dialog.Title>
              <Dialog.Description className="text-muted-foreground text-sm [overflow-wrap:anywhere]">
                {installed.from
                  ? t("appUpdate.completedDescription", {
                      from: installed.from,
                      version: installed.version,
                    })
                  : t("appUpdate.completedDescriptionUnknown", { version: installed.version })}
              </Dialog.Description>
            </div>
          </div>
          <Dialog.Close asChild>
            <Button size="icon-sm" variant="ghost" aria-label={t("appUpdate.completedClose")}>
              <X aria-hidden="true" />
            </Button>
          </Dialog.Close>
        </header>
        <div className="app-scroll min-h-0 flex-1 space-y-5 overflow-y-auto px-6 py-5">
          {installed.from ? (
            <section
              aria-label={t("appUpdate.releaseInfo")}
              className="bg-muted/40 flex flex-wrap items-center gap-x-3 gap-y-1.5 rounded-xl border px-4 py-3"
            >
              <span className="text-muted-foreground font-mono text-sm [overflow-wrap:anywhere]">
                {installed.from}
              </span>
              <ArrowRight className="text-muted-foreground size-4 shrink-0" aria-hidden="true" />
              <span className="text-primary font-mono text-sm font-semibold [overflow-wrap:anywhere]">
                {installed.version}
              </span>
            </section>
          ) : null}
          <section className="space-y-2">
            <h3 className="text-sm font-medium">{t("appUpdate.completedNotes")}</h3>
            <div className="bg-muted/20 rounded-xl border px-4 py-3">
              {notes ? (
                <Markdown source={withoutTopHeading(notes)} />
              ) : (
                <p className="text-muted-foreground text-sm">{t("appUpdate.completedNoNotes")}</p>
              )}
            </div>
          </section>
        </div>
        <footer className="shrink-0 border-t px-6 py-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-muted-foreground text-xs">{t("appUpdate.completedHint")}</p>
            <div className="ml-auto flex flex-wrap items-center gap-2">
              <Button size="sm" variant="outline" onClick={onOpenNotes}>
                <ScrollText aria-hidden="true" />
                {t("appUpdate.completedOpenNotes")}
              </Button>
              <Button size="sm" onClick={onConfirm}>
                {t("appUpdate.completedConfirm")}
              </Button>
            </div>
          </div>
        </footer>
      </Dialog.Content>
    </Dialog.Portal>
  )
}
