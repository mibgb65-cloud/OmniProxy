import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect } from "react"
import { Events } from "@wailsio/runtime"
import {
  createAccount,
  createAccounts,
  deleteAccount,
  listAccounts,
  setAccountEnabled,
  updateAccount,
  type CreateAccountInput,
  type CreateAccountsInput,
  type UpdateAccountInput,
} from "../../services/omniproxy"

const accountQueryKey = ["accounts"] as const

export function useAccounts() {
  const client = useQueryClient()
  useEffect(
    () =>
      Events.On("subscription:updated", () => {
        void client.invalidateQueries({ queryKey: accountQueryKey })
      }),
    [client],
  )
  return useQuery({ queryKey: accountQueryKey, queryFn: listAccounts })
}

export function useCreateAccount() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateAccountInput) => createAccount(input),
    onSuccess: () =>
      Promise.all([
        client.invalidateQueries({ queryKey: accountQueryKey }),
        client.invalidateQueries({ queryKey: ["model-catalog"] }),
        client.invalidateQueries({ queryKey: ["default-routes"] }),
      ]),
  })
}

export function useCreateAccounts() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateAccountsInput) => createAccounts(input),
    onSuccess: () =>
      Promise.all([
        client.invalidateQueries({ queryKey: accountQueryKey }),
        client.invalidateQueries({ queryKey: ["model-catalog"] }),
        client.invalidateQueries({ queryKey: ["default-routes"] }),
      ]),
  })
}

export function useUpdateAccount() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (input: UpdateAccountInput) => updateAccount(input),
    gcTime: 0,
    onSettled: async (_data, _error, input) => {
      await Promise.all([
        client.invalidateQueries({ queryKey: accountQueryKey }),
        client.invalidateQueries({ queryKey: ["account-key-preview", input.id] }),
        client.invalidateQueries({ queryKey: ["model-catalog"] }),
        client.invalidateQueries({ queryKey: ["default-routes"] }),
      ])
    },
  })
}

export function useSetAccountEnabled() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      setAccountEnabled(id, enabled),
    onSuccess: () =>
      Promise.all([
        client.invalidateQueries({ queryKey: accountQueryKey }),
        client.invalidateQueries({ queryKey: ["model-catalog"] }),
        client.invalidateQueries({ queryKey: ["default-routes"] }),
      ]),
  })
}

export function useDeleteAccount() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteAccount(id),
    // 清理密文失败时数据库记录也可能已删除，仍需刷新列表。
    onSettled: async (_data, _error, id) => {
      await Promise.all([
        client.invalidateQueries({ queryKey: accountQueryKey }),
        client.invalidateQueries({ queryKey: ["model-catalog"] }),
        client.invalidateQueries({ queryKey: ["default-routes"] }),
      ])
      client.removeQueries({ queryKey: ["account-key-preview", id] })
    },
  })
}
