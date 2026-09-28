import { useState } from "react"
import { useTranslation } from "react-i18next"
import type { AccountItem, CreateAccountInput } from "../../services/omniproxy"
import { useCreateAccount, useDeleteAccount, useSetAccountEnabled } from "./use-accounts"

export function useAccountActions() {
  const { t } = useTranslation()
  const create = useCreateAccount()
  const setEnabled = useSetAccountEnabled()
  const remove = useDeleteAccount()
  const [actionError, setActionError] = useState(false)

  const save = async (input: CreateAccountInput) => {
    await create.mutateAsync(input)
  }

  const toggle = (id: string, enabled: boolean) => {
    setActionError(false)
    void setEnabled.mutateAsync({ id, enabled }).catch(() => setActionError(true))
  }

  const deleteItem = (account: AccountItem) => {
    if (!window.confirm(t("accounts.deleteConfirm", { name: account.name }))) return
    setActionError(false)
    void remove.mutateAsync(account.id).catch(() => setActionError(true))
  }

  return {
    save,
    toggle,
    deleteItem,
    actionError,
    busy: setEnabled.isPending || remove.isPending,
  }
}
