import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  discoverProxyPorts,
  getProxySettings,
  saveProxySettings,
  setProviderProxyMode,
  testProxySettings,
  type ProxySaveInput,
} from "../../services/omniproxy"

const providerQueryKey = ["providers"] as const
const proxySettingsQueryKey = ["proxy-settings"] as const

export function useProxySettings(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: proxySettingsQueryKey,
    queryFn: () => getProxySettings(),
    // 弹窗常驻挂载（为了退出动画）时只有打开才读取设置。
    enabled: options.enabled ?? true,
  })
}

export function useSaveProxySettings() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (input: ProxySaveInput) => saveProxySettings(input),
    onSuccess: (settings) => {
      // 保存结果本身就是最新的脱敏配置，直接回填以省一次往返。
      client.setQueryData(proxySettingsQueryKey, settings)
    },
  })
}

export function useTestProxySettings() {
  return useMutation({ mutationFn: (input: ProxySaveInput) => testProxySettings(input) })
}

export function useDiscoverProxyPorts() {
  return useMutation({ mutationFn: () => discoverProxyPorts() })
}

export function useSetProviderProxyMode() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ id, mode }: { id: string; mode: string }) => setProviderProxyMode(id, mode),
    onSuccess: () => client.invalidateQueries({ queryKey: providerQueryKey }),
  })
}
