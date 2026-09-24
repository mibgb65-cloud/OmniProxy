import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  cleanStorage,
  getStorageInfo,
  selectDirectory,
  setDataDir,
  type CleanResult,
  type SetDataDirResult,
} from "../../services/omniproxy"

export type CleanTarget = "logs" | "database"

export const storageInfoQuery = queryOptions({
  queryKey: ["storage-info"],
  queryFn: getStorageInfo,
  staleTime: 5_000,
})

export function useStorageInfo() {
  return useQuery(storageInfoQuery)
}

export function useSelectDirectory() {
  // Wrapping keeps the call signature exact: TanStack Query passes an extra
  // context argument to mutationFn that the service layer must not see.
  return useMutation({ mutationFn: (title: string) => selectDirectory(title) })
}

export function useSetDataDir() {
  const queryClient = useQueryClient()
  return useMutation<SetDataDirResult, Error, string>({
    mutationFn: (path: string) => setDataDir(path),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["storage-info"] })
    },
  })
}

export function useCleanStorage() {
  const queryClient = useQueryClient()
  return useMutation<CleanResult, Error, CleanTarget>({
    mutationFn: (target: CleanTarget) => cleanStorage(target),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["storage-info"] })
    },
  })
}
