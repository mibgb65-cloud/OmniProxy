import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type {
  Entry,
  Query,
  Snapshot,
} from "../../bindings/omniproxy/internal/infrastructure/logger"

export type { Entry as LogEntry, Snapshot as LogSnapshot }

export function readLogs(query: Query): Promise<Snapshot> {
  return AppService.LogsRead(query)
}
export function getLogHead(): Promise<number> {
  return AppService.LogsHead()
}
export function exportLogs(ids: number[], language: "zh" | "en") {
  return AppService.LogsExport(ids, language)
}
export function openLogDirectory(): Promise<void> {
  return AppService.LogsOpenDirectory()
}
