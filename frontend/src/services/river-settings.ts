import { AppService } from "../../bindings/omniproxy/internal/transport/wails"

export const getRiverSettings = async () => AppService.GetRiverSettings()
export const setRiverAccountRotationMode = async (mode: "queue" | "balanced") =>
  AppService.SetRiverAccountRotationMode(mode)
