import { AppService } from "../../bindings/omniproxy/internal/transport/wails"

export type ProjectResource = "website" | "repository" | "disclaimer" | "terms" | "privacy"

export function openProjectResource(resource: ProjectResource): Promise<void> {
  return AppService.OpenProjectResource(resource)
}
