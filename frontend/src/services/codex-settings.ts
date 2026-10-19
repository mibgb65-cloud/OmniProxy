import { AppService } from "../../bindings/omniproxy/internal/transport/wails"

export const getCodexSettings = () => AppService.GetCodexSettings()
export const setCodexAccountRotationMode = (mode: string) =>
  AppService.SetCodexAccountRotationMode(mode)
export const setCodexWebSocketEnabled = (enabled: boolean) =>
  AppService.SetCodexWebSocketEnabled(enabled)
export const setCodexQuotaAutoActivation = (enabled: boolean) =>
  AppService.SetCodexQuotaAutoActivation(enabled)
export const getCodexQuotaActivationStatus = () => AppService.GetCodexQuotaActivationStatus()
export type CodexActivationStatus = Awaited<ReturnType<typeof getCodexQuotaActivationStatus>>
