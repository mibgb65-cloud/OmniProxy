import { AppService } from "../../bindings/omniproxy/internal/transport/wails"

export function getUpdatePreferences() {
  return AppService.GetUpdatePreferences()
}
export function setUpdateChannel(channel: string) {
  return AppService.SetUpdateChannel(channel)
}
export type {
  Snapshot as ApplicationUpdate,
  Transfer as UpdateTransfer,
} from "../../bindings/omniproxy/internal/application/updater"
export function getApplicationUpdate() {
  return AppService.ApplicationUpdate()
}
export function startApplicationUpdate(version: string) {
  return AppService.StartApplicationUpdate(version)
}
export function cancelApplicationUpdate() {
  return AppService.CancelApplicationUpdate()
}
export function installApplicationUpdate(version: string) {
  return AppService.InstallApplicationUpdate(version)
}
export function setAutomaticUpdateCheck(enabled: boolean) {
  return AppService.SetAutomaticUpdateCheck(enabled)
}
