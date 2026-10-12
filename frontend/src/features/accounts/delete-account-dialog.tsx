import { Trash2 } from "lucide-react"
import { AlertDialog } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { useAccountActions } from "./use-account-actions"

export function DeleteAccountDialog({
  actions,
}: {
  actions: ReturnType<typeof useAccountActions>
}) {
  const { t } = useTranslation()
  return (
    <AlertDialog.Root
      open={actions.deleteTarget !== null}
      onOpenChange={(open) => {
        if (!open) actions.cancelDelete()
      }}
    >
      <AlertDialog.Portal>
        <AlertDialog.Overlay className="motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 fixed inset-0 z-40 bg-black/50" />
        <AlertDialog.Content
          className="bg-card text-card-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 fixed top-1/2 left-1/2 z-50 w-[min(440px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-xl border p-6 shadow-xl outline-none"
          onEscapeKeyDown={(event) => {
            if (actions.deleting) event.preventDefault()
          }}
          onCloseAutoFocus={(event) => {
            event.preventDefault()
            actions.restoreDeleteFocus()
          }}
        >
          <div className="flex items-start gap-3">
            <span className="bg-destructive/10 text-destructive flex size-10 shrink-0 items-center justify-center rounded-lg">
              <Trash2 className="size-5" aria-hidden="true" />
            </span>
            <div className="min-w-0 space-y-2">
              <AlertDialog.Title className="text-base font-semibold">
                {t("accounts.deleteTitle")}
              </AlertDialog.Title>
              <AlertDialog.Description className="text-muted-foreground text-sm break-words">
                {t("accounts.deleteConfirm", { name: actions.deleteTarget?.name ?? "" })}
              </AlertDialog.Description>
              <p className="text-muted-foreground text-xs">
                {t(
                  actions.deleteTarget?.authMethod === "subscription_login"
                    ? "subscription.deleteNotice"
                    : "accounts.deleteNotice",
                )}
              </p>
            </div>
          </div>
          {actions.deleteError ? (
            <p role="alert" className="text-destructive mt-4 text-sm">
              {t("accounts.deleteFailed")}
            </p>
          ) : null}
          <div className="mt-6 flex justify-end gap-2">
            <AlertDialog.Cancel asChild>
              <Button
                type="button"
                variant="outline"
                disabled={actions.deleting}
                className="cursor-pointer"
              >
                {t("accounts.cancel")}
              </Button>
            </AlertDialog.Cancel>
            <AlertDialog.Action
              asChild
              onClick={(event) => {
                event.preventDefault()
                void actions.confirmDelete()
              }}
            >
              <Button
                type="button"
                variant="destructive"
                disabled={actions.deleting}
                className="cursor-pointer"
              >
                {actions.deleting ? t("accounts.deleting") : t("accounts.deleteAction")}
              </Button>
            </AlertDialog.Action>
          </div>
        </AlertDialog.Content>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  )
}
