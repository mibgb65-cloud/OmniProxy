import { useRef, useState } from "react"
import type {
  AccountItem,
  CreateAccountInput,
  CreateAccountsInput,
  UpdateAccountInput,
} from "../../services/omniproxy"
import {
  useCreateAccount,
  useCreateAccounts,
  useDeleteAccount,
  useSetAccountEnabled,
  useUpdateAccount,
} from "./use-accounts"

export function useAccountActions() {
  const create = useCreateAccount()
  const createBatch = useCreateAccounts()
  const setEnabled = useSetAccountEnabled()
  const remove = useDeleteAccount()
  const update = useUpdateAccount()
  const [actionError, setActionError] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<AccountItem | null>(null)
  const [deleteError, setDeleteError] = useState(false)
  const deletingRef = useRef(false)
  const deleteFocusRef = useRef<HTMLElement | null>(null)
  const fallbackFocusRef = useRef<HTMLElement | null>(null)
  const [editTarget, setEditTarget] = useState<AccountItem | null>(null)
  const [updateWarning, setUpdateWarning] = useState(false)
  const updatingRef = useRef(false)

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

  const rememberModalFocus = () => {
    const focused = document.activeElement instanceof HTMLElement ? document.activeElement : null
    deleteFocusRef.current = focused
    fallbackFocusRef.current =
      focused?.closest('[data-slot="card"]')?.querySelector('button[data-variant="outline"]') ??
      focused?.closest("section")?.querySelector('button[data-variant="outline"]') ??
      null
  }

  const deleteItem = (account: AccountItem) => {
    if (deletingRef.current || editTarget) return
    rememberModalFocus()
    setDeleteError(false)
    setDeleteTarget(account)
  }

  const editItem = (account: AccountItem) => {
    if (updatingRef.current || deleteTarget) return
    rememberModalFocus()
    setEditTarget(account)
  }

  const cancelEdit = () => {
    if (!updatingRef.current) setEditTarget(null)
  }

  const saveEdit = async (input: UpdateAccountInput) => {
    if (updatingRef.current) return
    updatingRef.current = true
    setUpdateWarning(false)
    try {
      const result = await update.mutateAsync(input)
      setUpdateWarning(result.cleanupWarning)
      setEditTarget(null)
    } finally {
      updatingRef.current = false
      // 提交结束清除 mutation 中的新密钥输入；失败时只在编辑表单保留以便重试。
      update.reset()
    }
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
    editItem,
    editTarget,
    cancelEdit,
    saveEdit,
    restoreEditFocus: restoreDeleteFocus,
    updateWarning,
    actionError,
    busy: setEnabled.isPending || remove.isPending || update.isPending,
  }
}
