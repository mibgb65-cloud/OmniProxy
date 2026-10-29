import { Dialog } from "radix-ui"
import { useLastValue } from "../../lib/use-last-value"
import { useOpenGeneration } from "../../lib/use-open-generation"
import type { useAccountActions } from "./use-account-actions"
import { AccountEditForm } from "./account-edit-form"

export function EditAccountDialog({ actions }: { actions: ReturnType<typeof useAccountActions> }) {
  const open = actions.editTarget !== null
  // 关闭动画期间仍需要账号数据渲染，保留最后一次打开的目标而不是立刻卸载。
  const target = useLastValue(actions.editTarget)
  // 每次重新打开换 key，草稿不会残留。
  const generation = useOpenGeneration(open)
  return (
    <Dialog.Root
      open={open}
      onOpenChange={(next) => {
        if (!next) actions.cancelEdit()
      }}
    >
      {target ? (
        <AccountEditForm
          key={generation}
          account={target}
          onSubmit={actions.saveEdit}
          onRestoreFocus={actions.restoreEditFocus}
        />
      ) : null}
    </Dialog.Root>
  )
}
