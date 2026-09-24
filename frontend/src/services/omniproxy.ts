import {
  AppService,
  type AppInfo,
  type CleanResult,
  type SetDataDirResult,
  type StorageInfo,
} from "../../bindings/omniproxy/internal/transport/wails"

export type { AppInfo, CleanResult, SetDataDirResult, StorageInfo }

/** getAppInfo returns the application runtime information from the Go backend. */
export function getAppInfo(): Promise<AppInfo> {
  return AppService.AppInfo()
}

/** ping is a liveness probe against the Go backend. */
export function ping(): Promise<string> {
  return AppService.Ping()
}

/** getStorageInfo returns data storage locations and their sizes. */
export function getStorageInfo(): Promise<StorageInfo> {
  return AppService.StorageInfo()
}

/**
 * setDataDir persists a new data directory. The change takes effect on the
 * next start; existing data is moved when the app shuts down.
 */
export function setDataDir(path: string): Promise<SetDataDirResult> {
  return AppService.SetDataDir(path)
}

/** selectDirectory opens the native folder picker; empty string = cancelled. */
export function selectDirectory(title: string): Promise<string> {
  return AppService.SelectDirectory(title)
}

/** cleanStorage runs a cleanup operation ("logs" | "database"). */
export function cleanStorage(target: "logs" | "database"): Promise<CleanResult> {
  return AppService.CleanStorage(target)
}
