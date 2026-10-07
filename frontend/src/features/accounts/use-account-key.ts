import { useMutation, useQuery } from "@tanstack/react-query"
import { copyAccountKey, getAccountKeyPreview } from "../../services/omniproxy"

export function useAccountKeyPreview(id: string) {
  return useQuery({
    queryKey: ["account-key-preview", id],
    queryFn: () => getAccountKeyPreview(id),
    staleTime: 60_000,
    retry: false,
  })
}

export function useCopyAccountKey(id: string) {
  return useMutation({ mutationFn: () => copyAccountKey(id), retry: false })
}
