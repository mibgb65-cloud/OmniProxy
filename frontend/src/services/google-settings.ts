import { AppService } from "../../bindings/omniproxy/internal/transport/wails"

export const getGoogleSettings = async () => AppService.GetGoogleSettings()
export const setGoogleAccountRotationMode = async (mode: "queue" | "balanced") =>
  AppService.SetGoogleAccountRotationMode(mode)
