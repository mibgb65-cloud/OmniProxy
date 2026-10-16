import { AppService } from "../../bindings/omniproxy/internal/transport/wails"

export const getStepFunSettings = () => AppService.GetStepFunSettings()
export const setStepFunPreferredBillingMode = (mode: string) =>
  AppService.SetStepFunPreferredBillingMode(mode)
