import { queryOptions, useQuery } from "@tanstack/react-query"
import { getAppInfo, type AppInfo } from "../services/omniproxy"

export const appInfoQuery = queryOptions({
  queryKey: ["app-info"],
  queryFn: getAppInfo,
  staleTime: Number.POSITIVE_INFINITY,
})

export function useAppInfo() {
  return useQuery(appInfoQuery)
}

export type { AppInfo }
