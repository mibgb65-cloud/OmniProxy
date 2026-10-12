import type { TranslationKey } from "../../i18n/translation"
import { useAccounts } from "../accounts/use-accounts"
import { gatewayErrorKey, gatewayOperationErrorKey } from "./gateway-errors"
import { getConfiguredProxyAddress } from "./gateway-preparation"
import { useGatewayRuntime } from "./use-gateway-runtime"

export function useSidebarGatewayStatus() {
  const gateway = useGatewayRuntime()
  const accounts = useAccounts()
  const port = gateway.info.data?.runtime.port
  const errorKey: TranslationKey | null = gateway.info.isError
    ? "gateway.status.loadFailed"
    : gateway.operationError
      ? gatewayOperationErrorKey(gateway.operationError)
      : gateway.state === "error" && gateway.info.data?.runtime.errorCode
        ? gatewayErrorKey(gateway.info.data.runtime.errorCode)
        : accounts.isError
          ? "gateway.sidebar.accountsFailed"
          : null

  return {
    gateway,
    port: gateway.info.isSuccess && getConfiguredProxyAddress(port) ? port : null,
    accountCount: accounts.isSuccess ? accounts.data.length : null,
    accountsLoading: accounts.isPending,
    errorKey,
  }
}
