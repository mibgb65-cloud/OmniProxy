import { AppService } from "../../bindings/omniproxy/internal/transport/wails"

export const getCodexSettings = () => AppService.GetCodexSettings()
export const setCodexWebSocketEnabled = (enabled: boolean) =>
  AppService.SetCodexWebSocketEnabled(enabled)
export const setCodexQuotaAutoActivation = (enabled: boolean) =>
  AppService.SetCodexQuotaAutoActivation(enabled)
export const getCodexQuotaActivationStatus = () => AppService.GetCodexQuotaActivationStatus()
