import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type {
  FailoverSettings,
  AccountHealth,
} from "../../bindings/omniproxy/internal/application/gateway"
export type {
  FailoverSettings,
  AccountHealth,
} from "../../bindings/omniproxy/internal/application/gateway"

export const getFailoverSettings = (): Promise<FailoverSettings> =>
  AppService.GatewayFailoverSettings()
export const setFailoverEnabled = (enabled: boolean): Promise<FailoverSettings> =>
  AppService.GatewaySetFailoverEnabled(enabled)
export const getFailoverHealth = (): Promise<AccountHealth[] | null> =>
  AppService.GatewayFailoverHealth()
