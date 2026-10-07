import { Dialog } from "radix-ui"
import type { useAccountActions } from "./use-account-actions"
import { AccountEditForm } from "./account-edit-form"

export function EditAccountDialog({ actions }: { actions: ReturnType<typeof useAccountActions> }) {
  return (
    <Dialog.Root
      open={actions.editTarget !== null}
      onOpenChange={(open) => {
        if (!open) actions.cancelEdit()
      }}
    >
      {actions.editTarget ? (
        <AccountEditForm
          account={actions.editTarget}
          onSubmit={actions.saveEdit}
          onRestoreFocus={actions.restoreEditFocus}
        />
      ) : null}
    </Dialog.Root>
  )
}
