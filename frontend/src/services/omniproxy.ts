import {
  AppService,
  type AppInfo,
  type CleanResult,
  type DesktopSettings,
  type RuntimeInfo,
  type SetDataDirResult,
  type StorageInfo,
  type UpdateStatus,
} from "../../bindings/omniproxy/internal/transport/wails"

export type {
  AppInfo,
  CleanResult,
  DesktopSettings,
  RuntimeInfo,
  SetDataDirResult,
  StorageInfo,
  UpdateStatus,
}

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

/** getDesktopSettings reads the current OS autostart registration and close preference. */
export function getDesktopSettings(): Promise<DesktopSettings> {
  return AppService.DesktopSettings()
}

/** setAutoStart registers or removes this profile from the user's login startup. */
export function setAutoStart(enabled: boolean): Promise<boolean> {
  return AppService.SetAutoStart(enabled)
}

/** setCloseBehavior persists how closing the main window should behave. */
export function setCloseBehavior(value: "exit" | "tray"): Promise<string> {
  return AppService.SetCloseBehavior(value)
}

/** getRuntimeInfo returns application and runtime details for the About page. */
export function getRuntimeInfo(): Promise<RuntimeInfo> {
  return AppService.RuntimeInfo()
}

/** checkForUpdate compares the current version against the latest release. */
export function checkForUpdate(): Promise<UpdateStatus> {
  return AppService.CheckForUpdate()
}
