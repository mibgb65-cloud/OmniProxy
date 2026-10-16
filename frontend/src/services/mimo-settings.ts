import { AppService } from "../../bindings/omniproxy/internal/transport/wails"

export const getMiMoSettings = () => AppService.GetMiMoSettings()
export const setMiMoPreferredBillingMode = (mode: string) =>
  AppService.SetMiMoPreferredBillingMode(mode)
