import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useAppInfo } from "../../hooks/use-app-info"
import { listAccounts, listProviders } from "../../services/omniproxy"
import {
  getConfiguredProxyAddress,
  getConfiguredResources,
  type PreparationState,
} from "./gateway-preparation"
import { useGatewayRuntime } from "./use-gateway-runtime"
import { withGatewayFeedback } from "./gateway-action-feedback"

export function useGatewayPreparation() {
  const client = useQueryClient()
  const runtime = useAppInfo()
  const gateway = useGatewayRuntime()
  // 共享只读查询键，让账号、提供商页面的缓存失效自然同步到接入检查。
  const providers = useQuery({ queryKey: ["providers"], queryFn: listProviders })
  const accounts = useQuery({ queryKey: ["accounts"], queryFn: listAccounts })
  const proxyAddress = runtime.isSuccess ? getConfiguredProxyAddress(runtime.data.proxyPort) : null
  const activeAddress = gateway.isRunning
    ? getConfiguredProxyAddress(gateway.info.data?.runtime.port)
    : null
  const address = (activeAddress ?? proxyAddress) ? `${activeAddress ?? proxyAddress}/v1` : null
  const supportedProviders =
    providers.data?.filter((item) =>
      ["river", "deepseek", "atria", "google", "mimo", "stepfun", "kimi"].includes(item.kind),
    ) ?? []
  const resources =
    providers.isSuccess && accounts.isSuccess
      ? getConfiguredResources(
          supportedProviders,
          accounts.data.filter((item) => item.authMethod !== "subscription_login"),
        )
      : null
  const providerCount = providers.isSuccess
    ? supportedProviders.filter((item) => item.enabled).length
    : null

  const runtimeState: PreparationState = runtime.isPending
    ? "loading"
    : runtime.isError || !address
      ? "error"
      : "configured"
  const providerState: PreparationState = providers.isPending
    ? "loading"
    : providers.isError
      ? "error"
      : providerCount
        ? "configured"
        : "missing"
  const accountState: PreparationState =
    accounts.isPending || providers.isPending
      ? "loading"
      : accounts.isError || providers.isError
        ? "error"
        : resources?.accountCount
          ? "configured"
          : "missing"

  const refresh = useMutation({
    mutationFn: () =>
      withGatewayFeedback(async () => {
        await Promise.all([
          runtime.refetch(),
          providers.refetch(),
          accounts.refetch(),
          gateway.info.refetch(),
          client.refetchQueries({ queryKey: ["agent-clients"], type: "active" }),
        ])
      }),
  })

  return {
    runtime,
    gateway,
    address,
    providerCount,
    accountCount: resources?.accountCount ?? null,
    runtimeState,
    providerState,
    accountState,
    checksLoading:
      runtime.isPending || providers.isPending || accounts.isPending || gateway.info.isPending,
    // 后台状态同步不属于用户手动刷新，不切换按钮的文案和动画。
    isRefreshing: refresh.isPending,
    refresh: () => {
      if (!refresh.isPending) refresh.mutate()
    },
  }
}

export type GatewayPreparation = ReturnType<typeof useGatewayPreparation>
