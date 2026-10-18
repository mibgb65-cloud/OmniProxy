import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { checkForUpdate, getRuntimeInfo } from "../../services/omniproxy"

export const runtimeInfoQuery = queryOptions({
  queryKey: ["runtime-info"],
  queryFn: getRuntimeInfo,
  staleTime: 30_000,
})

export function useRuntimeInfo() {
  return useQuery(runtimeInfoQuery)
}

export function useCheckForUpdate() {
  // The check must run on demand (button), never on mount: it hits the
  // network and is rate-limited by GitHub.
  const client = useQueryClient()
  return useMutation({
    mutationFn: () => checkForUpdate(),
    onSuccess: () => client.invalidateQueries({ queryKey: ["application-update"] }),
  })
}
