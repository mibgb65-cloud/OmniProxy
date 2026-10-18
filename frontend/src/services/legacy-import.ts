import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
export type { Entry as LegacyEntry } from "../../bindings/omniproxy/internal/application/legacyimport"

export async function previewLegacyImport(directory: string) {
  const result = await AppService.PreviewLegacyImport(directory)
  return { ...result, entries: result.entries ?? [] }
}
export async function importLegacyData(directory: string, snapshot: string) {
  const result = await AppService.ImportLegacyData(directory, snapshot)
  return { ...result, entries: result.entries ?? [] }
}
