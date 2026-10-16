import { AppService } from "../../bindings/omniproxy/internal/transport/wails"

export const getKimiSettings = () => AppService.GetKimiSettings()
export const setKimiPreferredBillingMode = (mode: string) =>
  AppService.SetKimiPreferredBillingMode(mode)
export const setKimiPreferredRegion = (region: string) => AppService.SetKimiPreferredRegion(region)
export const openKimiCodeConsole = (id: string) => AppService.OpenKimiCodeConsole(id)
