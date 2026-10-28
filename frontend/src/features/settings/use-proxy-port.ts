import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  getProxyPort,
  setProxyPort,
  type ProxyPortState,
  type ProxyPortUpdate,
} from "../../services/omniproxy"

export const proxyPortQuery = queryOptions({
  queryKey: ["proxy-port"],
  queryFn: getProxyPort,
  staleTime: 5_000,
})

export function useProxyPort() {
  return useQuery(proxyPortQuery)
}

export function useSetProxyPort() {
  const queryClient = useQueryClient()
  return useMutation<ProxyPortUpdate, Error, number>({
    mutationFn: (port) => setProxyPort(port),
    onSuccess: (update) => {
      queryClient.setQueryData<ProxyPortState>(proxyPortQuery.queryKey, update.state)
      // 端口同时出现在应用信息、关于页和网关页，全部重新读取避免各自显示旧值。
      void queryClient.invalidateQueries({ queryKey: ["app-info"] })
      void queryClient.invalidateQueries({ queryKey: ["runtime-info"] })
      void queryClient.invalidateQueries({ queryKey: ["gateway-runtime"] })
      void queryClient.invalidateQueries({ queryKey: ["agent-clients"] })
    },
  })
}
