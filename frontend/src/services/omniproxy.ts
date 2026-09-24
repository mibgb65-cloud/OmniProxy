import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type { AppInfo as AppInfoModel } from "../../bindings/omniproxy/internal/transport/wails"

export type AppInfo = AppInfoModel

/**
 * getAppInfo returns the application runtime information from the Go backend
 * through the Wails typed binding.
 */
export function getAppInfo(): Promise<AppInfo> {
  return AppService.AppInfo()
}

/** ping is a liveness probe against the Go backend. */
export function ping(): Promise<string> {
  return AppService.Ping()
}
