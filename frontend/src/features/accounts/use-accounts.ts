import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  createAccount,
  deleteAccount,
  listAccounts,
  setAccountEnabled,
  type CreateAccountInput,
} from "../../services/omniproxy"

const accountQueryKey = ["accounts"] as const

export function useAccounts() {
  return useQuery({ queryKey: accountQueryKey, queryFn: listAccounts })
}

export function useCreateAccount() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateAccountInput) => createAccount(input),
    onSuccess: () => client.invalidateQueries({ queryKey: accountQueryKey }),
  })
}

export function useSetAccountEnabled() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      setAccountEnabled(id, enabled),
    onSuccess: () => client.invalidateQueries({ queryKey: accountQueryKey }),
  })
}

export function useDeleteAccount() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteAccount(id),
    // 清理密文失败时数据库记录也可能已删除，仍需刷新列表。
    onSettled: () => client.invalidateQueries({ queryKey: accountQueryKey }),
  })
}
