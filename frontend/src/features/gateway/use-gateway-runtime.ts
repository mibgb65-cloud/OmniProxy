import {
  useIsMutating,
  useMutation,
  useMutationState,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import { useEffect } from "react"
import type { TranslationKey } from "../../i18n/translation"
import { getGatewayInfo, startGateway, stopGateway, type GatewayInfo } from "../../services/gateway"
import { withGatewayFeedback } from "./gateway-action-feedback"

const gatewayQueryKey = ["gateway-runtime"] as const
const gatewayStartKey = [...gatewayQueryKey, "start"] as const
const gatewayStopKey = [...gatewayQueryKey, "stop"] as const

export function useGatewayRuntime() {
  const client = useQueryClient()
  const info = useQuery({ queryKey: gatewayQueryKey, queryFn: getGatewayInfo })
  const changeRuntime = (operation: () => Promise<GatewayInfo>) =>
    withGatewayFeedback(async () => {
      const result = await operation()
      client.setQueryData(gatewayQueryKey, result)
      return result
    })
  const start = useMutation({
    mutationKey: gatewayStartKey,
    mutationFn: () => changeRuntime(startGateway),
    onError: () => client.invalidateQueries({ queryKey: gatewayQueryKey }),
  })
  const stop = useMutation({
    mutationKey: gatewayStopKey,
    mutationFn: () => changeRuntime(stopGateway),
    onError: () => client.invalidateQueries({ queryKey: gatewayQueryKey }),
  })
  // 侧栏和网关页共用操作中状态，避免不同入口重复提交启停。
  const isStarting = useIsMutating({ mutationKey: gatewayStartKey }) > 0
  const isStopping = useIsMutating({ mutationKey: gatewayStopKey }) > 0
  const operationError =
    useMutationState({
      filters: { mutationKey: gatewayQueryKey },
      select: (mutation) => (mutation.state.status === "error" ? mutation.state.error : null),
    }).at(-1) ?? null

  useEffect(
    () =>
      Events.On("gateway:status", () => {
        void client.invalidateQueries({ queryKey: gatewayQueryKey })
      }),
    [client],
  )

  const state =
    info.isSuccess && ["running", "stopped", "stopping", "error"].includes(info.data.runtime.state)
      ? info.data.runtime.state
      : info.isPending
        ? "loading"
        : "unknown"
  const statusKey: TranslationKey = isStarting
    ? "gateway.status.starting"
    : isStopping || state === "stopping"
      ? "gateway.status.stopping"
      : state === "running"
        ? "gateway.status.running"
        : state === "stopped"
          ? "gateway.status.stopped"
          : state === "error"
            ? "gateway.status.error"
            : state === "loading"
              ? "gateway.loading"
              : "gateway.status.unknown"

  function toggle() {
    if (
      client.isMutating({ mutationKey: gatewayQueryKey }) ||
      state === "stopping" ||
      !info.isSuccess
    )
      return
    start.reset()
    stop.reset()
    if (state === "running") stop.mutate()
    else start.mutate()
  }

  return {
    info,
    start,
    stop,
    state,
    statusKey,
    operationError,
    toggle,
    isRunning: state === "running",
    controlsBusy: isStarting || isStopping || state === "stopping",
  }
}
