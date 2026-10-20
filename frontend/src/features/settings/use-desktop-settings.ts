import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  getDesktopSettings,
  setAutoStart,
  setCloseBehavior,
  setGatewayAutoStart,
  type DesktopSettings,
} from "../../services/omniproxy"

export type CloseBehavior = "exit" | "tray"

export const desktopSettingsQuery = queryOptions({
  queryKey: ["desktop-settings"],
  queryFn: getDesktopSettings,
  staleTime: 5_000,
})

export function useDesktopSettings() {
  return useQuery(desktopSettingsQuery)
}

export function useSetAutoStart() {
  const queryClient = useQueryClient()
  return useMutation<boolean, Error, boolean>({
    mutationFn: (enabled) => setAutoStart(enabled),
    onSuccess: (enabled) => {
      queryClient.setQueryData<DesktopSettings>(desktopSettingsQuery.queryKey, (old) =>
        old ? { ...old, autoStart: enabled } : old,
      )
      void queryClient.invalidateQueries({ queryKey: desktopSettingsQuery.queryKey })
    },
  })
}

export function useSetCloseBehavior() {
  const queryClient = useQueryClient()
  return useMutation<string, Error, CloseBehavior>({
    mutationFn: (value) => setCloseBehavior(value),
    onSuccess: (value) => {
      queryClient.setQueryData<DesktopSettings>(desktopSettingsQuery.queryKey, (old) =>
        old ? { ...old, closeBehavior: value } : old,
      )
      void queryClient.invalidateQueries({ queryKey: desktopSettingsQuery.queryKey })
    },
  })
}

export function useSetGatewayAutoStart() {
  const queryClient = useQueryClient()
  return useMutation<boolean, Error, boolean>({
    mutationFn: (enabled) => setGatewayAutoStart(enabled),
    onSuccess: (enabled) => {
      queryClient.setQueryData<DesktopSettings>(desktopSettingsQuery.queryKey, (old) =>
        old ? { ...old, gatewayAutoStart: enabled } : old,
      )
      void queryClient.invalidateQueries({ queryKey: desktopSettingsQuery.queryKey })
    },
  })
}
