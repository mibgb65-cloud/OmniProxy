import { readLogs, type LogEntry, type LogSnapshot } from "../../services/logs"

export const logQueryKey = ["live-logs"] as const
export const logLevels = ["DEBUG", "INFO", "WARN", "ERROR"] as const
export type LogLevel = (typeof logLevels)[number]
export type NormalizedLogSnapshot = Omit<LogSnapshot, "entries" | "modules"> & {
  entries: LogEntry[]
  modules: string[]
}
export type LogData = Omit<NormalizedLogSnapshot, "nextCursor" | "hasMore" | "until"> & {
  cursor: number
  clearAfter: number
}

export function isLogLevel(value: string): value is LogLevel {
  return logLevels.some((level) => level === value)
}

function validateSnapshot(snapshot: LogSnapshot): NormalizedLogSnapshot {
  if (
    !Number.isSafeInteger(snapshot.latestId) ||
    snapshot.latestId < 0 ||
    !Number.isSafeInteger(snapshot.nextCursor) ||
    snapshot.nextCursor < 0 ||
    snapshot.nextCursor > snapshot.latestId ||
    !Number.isInteger(snapshot.capacity) ||
    snapshot.capacity < 1 ||
    snapshot.capacity > 5000 ||
    !Array.isArray(snapshot.entries) ||
    snapshot.entries.length > 200 ||
    snapshot.entries.some(
      (entry) =>
        !Number.isSafeInteger(entry.id) ||
        entry.id < 1 ||
        entry.id > snapshot.latestId ||
        !isLogLevel(entry.level),
    )
  )
    throw new Error("logs: invalid_snapshot")
  return { ...snapshot, entries: snapshot.entries, modules: snapshot.modules ?? [] }
}

/** 固定读取上界，通知只负责唤醒下一批，避免高频写入导致无限追赶。 */
export async function readLogRange(after: number, signal?: AbortSignal) {
  let cursor = after
  let until = 0
  let entries: LogEntry[] = []
  let dropped = false
  let reset = false
  for (let batch = 0; batch < 22; batch++) {
    if (signal?.aborted) throw new DOMException("Aborted", "AbortError")
    const snapshot = validateSnapshot(await readLogs({ after: cursor, until, limit: 200 }))
    if (snapshot.reset) {
      entries = []
      reset = true
    }
    entries.push(...snapshot.entries)
    dropped ||= snapshot.dropped
    if (snapshot.hasMore && !snapshot.reset && snapshot.nextCursor <= cursor)
      throw new Error("logs: invalid_cursor")
    cursor = snapshot.nextCursor
    until = snapshot.until
    if (!snapshot.hasMore) return { ...snapshot, entries, dropped, reset }
  }
  throw new Error("logs: invalid_range")
}

export function mergeLogData(
  previous: LogData | undefined,
  snapshot: NormalizedLogSnapshot,
): LogData {
  const clearAfter = snapshot.reset ? 0 : (previous?.clearAfter ?? 0)
  const byId = new Map<number, LogEntry>()
  if (!snapshot.reset) for (const entry of previous?.entries ?? []) byId.set(entry.id, entry)
  for (const entry of snapshot.entries) byId.set(entry.id, entry)
  const entries = [...byId.values()]
    .filter((entry) => entry.id > clearAfter)
    .sort((a, b) => a.id - b.id)
    .slice(-snapshot.capacity)
  return {
    ...snapshot,
    entries,
    cursor: snapshot.reset
      ? snapshot.nextCursor
      : Math.max(snapshot.nextCursor, previous?.cursor ?? 0),
    clearAfter,
    dropped: snapshot.reset ? snapshot.dropped : snapshot.dropped || !!previous?.dropped,
  }
}

export function filterLogs(
  entries: LogEntry[],
  levels: LogLevel[],
  module: string,
  keyword: string,
) {
  const search = keyword.trim().toLocaleLowerCase()
  return entries.filter(
    (entry) =>
      levels.some((level) => level === entry.level) &&
      (!module || entry.module === module) &&
      (!search ||
        [entry.message, entry.module, ...Object.values(entry.fields ?? {})]
          .join(" ")
          .toLocaleLowerCase()
          .includes(search)),
  )
}

export function formatLog(entry: LogEntry) {
  return JSON.stringify({
    id: entry.id,
    time: entry.time,
    level: entry.level,
    module: entry.module,
    message: entry.message,
    fields: entry.fields,
  })
}
