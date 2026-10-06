import { useRef, useState } from "react"
import type { AccountItem, CreateAccountInput, CreateAccountsInput } from "../../services/omniproxy"
import {
  useCreateAccount,
  useCreateAccounts,
  useDeleteAccount,
  useSetAccountEnabled,
} from "./use-accounts"

export function useAccountActions() {
  const create = useCreateAccount()
  const createBatch = useCreateAccounts()
  const setEnabled = useSetAccountEnabled()
  const remove = useDeleteAccount()
  const [actionError, setActionError] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<AccountItem | null>(null)
  const [deleteError, setDeleteError] = useState(false)
  const deletingRef = useRef(false)
  const deleteFocusRef = useRef<HTMLElement | null>(null)
  const fallbackFocusRef = useRef<HTMLElement | null>(null)

  const save = async (input: CreateAccountInput) => {
    await create.mutateAsync(input)
  }

  const saveBatch = async (input: CreateAccountsInput) => {
    await createBatch.mutateAsync(input)
  }

  const toggle = (id: string, enabled: boolean) => {
    setActionError(false)
    void setEnabled.mutateAsync({ id, enabled }).catch(() => setActionError(true))
  }

  const deleteItem = (account: AccountItem) => {
    if (deletingRef.current) return
    const focused = document.activeElement instanceof HTMLElement ? document.activeElement : null
    deleteFocusRef.current = focused
    fallbackFocusRef.current =
      focused?.closest('[data-slot="card"]')?.querySelector('button[data-variant="outline"]') ??
      focused?.closest("section")?.querySelector('button[data-variant="outline"]') ??
      null
    setDeleteError(false)
    setDeleteTarget(account)
  }

  const cancelDelete = () => {
    if (deletingRef.current) return
    setDeleteTarget(null)
    setDeleteError(false)
  }

  const confirmDelete = async () => {
    if (!deleteTarget || deletingRef.current) return
    deletingRef.current = true
    setDeleteError(false)
    try {
      await remove.mutateAsync(deleteTarget.id)
      setDeleteTarget(null)
    } catch {
      setDeleteError(true)
    } finally {
      deletingRef.current = false
    }
  }

  const restoreDeleteFocus = () => {
    const target = deleteFocusRef.current?.isConnected
      ? deleteFocusRef.current
      : fallbackFocusRef.current
    if (target?.isConnected) target.focus()
  }

  return {
    save,
    saveBatch,
    toggle,
    deleteItem,
    deleteTarget,
    deleteError,
    confirmDelete,
    cancelDelete,
    restoreDeleteFocus,
    deleting: remove.isPending,
    actionError,
    busy: setEnabled.isPending || remove.isPending,
  }
}
