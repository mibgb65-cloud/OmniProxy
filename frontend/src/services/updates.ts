import { AppService } from "../../bindings/omniproxy/internal/transport/wails"

export function getUpdatePreferences() {
  return AppService.GetUpdatePreferences()
}
export function setUpdateChannel(channel: string) {
  return AppService.SetUpdateChannel(channel)
}
