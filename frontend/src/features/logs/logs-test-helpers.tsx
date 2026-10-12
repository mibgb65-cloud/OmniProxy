import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render } from "@testing-library/react"
import { vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { readLogs, type LogEntry } from "../../services/logs"
import { LogsView } from "./logs-view"

export function makeEntry(
  id: number,
  level = "INFO",
  module = "gateway",
  message = `event ${id}`,
): LogEntry {
  return {
    id,
    level,
    module,
    message,
    time: "2026-10-08T03:00:00.123Z",
    fields: { code: "ok", model: "deepseek-flash" },
  }
}
export function mockLogReader(getEntries: () => LogEntry[]) {
  vi.mocked(readLogs).mockImplementation(async ({ after, until, limit }) => {
    const all = getEntries()
    const latestId = all.at(-1)?.id ?? 0
    const bound = until || latestId
    const matching = all.filter((entry) => entry.id > after && entry.id <= bound)
    const entries = matching.slice(0, limit)
    return {
      entries,
      nextCursor: matching.length > limit ? entries.at(-1)!.id : bound,
      latestId,
      oldestId: all[0]?.id ?? 0,
      until: bound,
      capacity: 512,
      modules: [...new Set(all.map((entry) => entry.module))],
      hasMore: matching.length > limit,
      dropped: false,
      reset: false,
      recordedLevel: "DEBUG",
    }
  })
}
export function renderLogs(
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } }),
) {
  return {
    ...render(
      <I18nProvider>
        <QueryClientProvider client={client}>
          <LogsView />
        </QueryClientProvider>
      </I18nProvider>,
    ),
    client,
  }
}
