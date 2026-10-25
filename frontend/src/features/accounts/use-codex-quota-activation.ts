import { useQuery, useQueryClient } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import { useEffect } from "react"
import {
  getCodexQuotaActivationStatus,
  getCodexSettings,
  type CodexActivationStatus,
} from "../../services/codex-settings"

export const codexActivationStatusKey = ["codex-quota-activation"] as const

// 后台每轮检测结束都会推送；低频轮询只兜底窗口隐藏或事件丢失时错过的推送。
const fallbackInterval = 30_000

// 自动激活状态是应用级共享查询：任何页面都能自动刷新，不必先打开设置面板。
export function useCodexQuotaActivation() {
  const client = useQueryClient()
  const settings = useQuery({ queryKey: ["codex-settings"], queryFn: getCodexSettings })
  const enabled = settings.data?.autoQuotaActivation === true
  useEffect(
    () =>
      Events.On("subscription:activation-updated", (event) => {
        const status = event?.data as CodexActivationStatus | undefined
        if (status) client.setQueryData(codexActivationStatusKey, status)
      }),
    [client],
  )
  const status = useQuery({
    queryKey: codexActivationStatusKey,
    queryFn: getCodexQuotaActivationStatus,
    enabled,
    retry: false,
    refetchInterval: enabled ? fallbackInterval : false,
  })
  return { settings, status, enabled }
}
