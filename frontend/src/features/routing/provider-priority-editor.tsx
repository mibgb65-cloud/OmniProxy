import { ArrowDown, ArrowUp, X } from "lucide-react"
import { Dialog } from "radix-ui"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { DefaultModelRoute } from "../../services/routing"
import { moveProviderPriority, providerPriorityChoices } from "./provider-priority"

export function ProviderPriorityEditor({
  item,
  onSave,
  onClose,
  onRestoreFocus,
}: {
  item: DefaultModelRoute
  onSave: (providerIds: string[]) => Promise<void>
  onClose: () => void
  onRestoreFocus: () => void
}) {
  const { t } = useTranslation()
  const [entries, setEntries] = useState(() => providerPriorityChoices(item))
  const [saving, setSaving] = useState(false)
  const [failed, setFailed] = useState(false)
  async function save(ids: string[]) {
    if (saving || ids.length > 100) return
    setSaving(true)
    setFailed(false)
    try {
      await onSave(ids)
      onClose()
    } catch {
      setFailed(true)
    } finally {
      setSaving(false)
    }
  }
  return (
    <Dialog.Root
      open
      onOpenChange={(open) => {
        if (!open && !saving) onClose()
      }}
    >
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/50" />
        <Dialog.Content
          className="bg-card text-card-foreground fixed top-1/2 left-1/2 z-50 max-h-[calc(100dvh-2rem)] w-[min(520px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border p-6 shadow-xl outline-none"
          onCloseAutoFocus={(event) => {
            event.preventDefault()
            onRestoreFocus()
          }}
          onEscapeKeyDown={(event) => {
            if (saving) event.preventDefault()
          }}
          onInteractOutside={(event) => {
            if (saving) event.preventDefault()
          }}
        >
          <div className="flex items-start justify-between gap-3">
            <Dialog.Title className="min-w-0 text-base font-semibold wrap-anywhere">
              {t("routing.priority.title", { model: item.model })}
            </Dialog.Title>
            <Button
              variant="ghost"
              size="icon"
              aria-label={t("routing.priority.close")}
              disabled={saving}
              onClick={onClose}
            >
              <X aria-hidden="true" />
            </Button>
          </div>
          <Dialog.Description className="text-muted-foreground mt-2 text-sm leading-5">
            {t("routing.priority.description")}
          </Dialog.Description>
          <ol className="my-4 space-y-2" aria-label={t("routing.priority.order")}>
            {entries.map((entry, index) => (
              <li key={entry.id} className="flex min-w-0 items-center gap-2 rounded-lg border p-2">
                <span className="bg-muted flex size-7 shrink-0 items-center justify-center rounded-md text-xs font-medium">
                  {index + 1}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium wrap-anywhere">{entry.name}</p>
                  <p className="text-muted-foreground text-xs wrap-anywhere">{entry.id}</p>
                  {!entry.available ? (
                    <p className="text-xs text-amber-700 dark:text-amber-400">
                      {t("routing.priority.unavailable")}
                    </p>
                  ) : null}
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-11"
                  aria-label={t("routing.priority.up", { provider: entry.name })}
                  disabled={saving || index === 0}
                  onClick={() => setEntries(moveProviderPriority(entries, index, -1))}
                >
                  <ArrowUp aria-hidden="true" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-11"
                  aria-label={t("routing.priority.down", { provider: entry.name })}
                  disabled={saving || index === entries.length - 1}
                  onClick={() => setEntries(moveProviderPriority(entries, index, 1))}
                >
                  <ArrowDown aria-hidden="true" />
                </Button>
                {!entry.available ? (
                  <Button
                    variant="ghost"
                    size="icon"
                    className="size-11"
                    aria-label={t("routing.priority.remove", { provider: entry.name })}
                    disabled={saving}
                    onClick={() => setEntries(entries.filter((value) => value.id !== entry.id))}
                  >
                    <X aria-hidden="true" />
                  </Button>
                ) : null}
              </li>
            ))}
          </ol>
          {entries.length > 100 ? (
            <p role="alert" className="text-destructive mb-3 text-sm">
              {t("routing.priority.limit")}
            </p>
          ) : null}
          {failed ? (
            <p role="alert" className="text-destructive mb-3 text-sm">
              {t("routing.priority.saveFailed")}
            </p>
          ) : null}
          <div className="flex flex-wrap justify-end gap-2">
            <Button variant="ghost" disabled={saving} onClick={() => void save([])}>
              {t("routing.priority.automatic")}
            </Button>
            <Button variant="outline" disabled={saving} onClick={onClose}>
              {t("routing.cancel")}
            </Button>
            <Button
              disabled={saving || entries.length > 100}
              onClick={() => void save(entries.map((entry) => entry.id))}
            >
              {t(saving ? "routing.saving" : "routing.priority.save")}
            </Button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
