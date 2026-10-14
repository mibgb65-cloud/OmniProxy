import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type {
  BindingInput,
  Resource,
} from "../../bindings/omniproxy/internal/application/googlequota"

export type GoogleResource = Resource
export const readGoogleResources = () => AppService.GoogleResources()
export const setGoogleProject = (input: BindingInput) => AppService.SetGoogleProject(input)
export const openGoogleConsole = (page: "usage" | "billing") => AppService.OpenGoogleConsole(page)

export function googleError(error: unknown): "changed" | "invalid" | "failed" {
  const message = error instanceof Error ? error.message : String(error)
  if (message === "google: account_changed") return "changed"
  if (message === "google: invalid_input") return "invalid"
  return "failed"
}
