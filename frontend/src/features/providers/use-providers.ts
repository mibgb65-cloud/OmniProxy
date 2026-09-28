import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  createProvider,
  deleteProvider,
  getProviderCatalog,
  listProviders,
  probeProvider,
  setProviderEnabled,
  updateProvider,
  type CreateProviderInput,
  type UpdateProviderInput,
} from "../../services/omniproxy"

const providerQueryKey = ["providers"] as const

export function useProviderCatalog() {
  return useQuery({
    queryKey: ["provider-catalog"],
    queryFn: getProviderCatalog,
    staleTime: Infinity,
  })
}

export function useProviders() {
  return useQuery({ queryKey: providerQueryKey, queryFn: listProviders })
}

export function useCreateProvider() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateProviderInput) => createProvider(input),
    onSuccess: () => client.invalidateQueries({ queryKey: providerQueryKey }),
  })
}

export function useUpdateProvider() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateProviderInput }) =>
      updateProvider(id, input),
    onSuccess: () => client.invalidateQueries({ queryKey: providerQueryKey }),
  })
}

export function useDeleteProvider() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteProvider(id),
    onSuccess: () => client.invalidateQueries({ queryKey: providerQueryKey }),
  })
}

export function useSetProviderEnabled() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      setProviderEnabled(id, enabled),
    onSuccess: () => client.invalidateQueries({ queryKey: providerQueryKey }),
  })
}

export function useProbeProvider() {
  return useMutation({ mutationFn: (id: string) => probeProvider(id) })
}
