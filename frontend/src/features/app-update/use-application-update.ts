import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  getApplicationUpdate,
  startApplicationUpdate,
  cancelApplicationUpdate,
  installApplicationUpdate,
  type ApplicationUpdate,
} from "../../services/updates"

export const applicationUpdateKey = ["application-update"] as const

export function useApplicationUpdate() {
  return useQuery({
    queryKey: applicationUpdateKey,
    queryFn: getApplicationUpdate,
    staleTime: 10_000,
    refetchInterval: (query) =>
      ["downloading", "verifying", "installing"].includes(query.state.data?.download.state ?? "")
        ? 500
        : 15_000,
  })
}

export function useApplicationUpdateActions() {
  const client = useQueryClient()
  const accept = (state: ApplicationUpdate) => client.setQueryData(applicationUpdateKey, state)
  const download = useMutation({
    mutationFn: (version: string) => startApplicationUpdate(version),
    onSuccess: accept,
  })
  const cancel = useMutation({ mutationFn: () => cancelApplicationUpdate(), onSuccess: accept })
  const install = useMutation({
    mutationFn: (version: string) => installApplicationUpdate(version),
    onSuccess: accept,
    onError: () => client.invalidateQueries({ queryKey: applicationUpdateKey }),
  })
  return { download, cancel, install }
}
