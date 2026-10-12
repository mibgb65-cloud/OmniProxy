import { beforeEach, describe, expect, it, vi } from "vitest"
import { readLogs, type LogEntry } from "../../services/logs"
import { filterLogs, mergeLogData, readLogRange, type NormalizedLogSnapshot } from "./log-stream"

vi.mock("../../services/logs", () => ({ readLogs: vi.fn() }))
const entry = (id: number, level = "INFO"): LogEntry => ({
  id,
  level,
  time: "2026-10-08T03:00:00Z",
  module: "gateway",
  message: `message ${id}`,
  fields: { code: "ok" },
})
const snapshot = (
  entries: LogEntry[],
  overrides: Partial<NormalizedLogSnapshot> = {},
): NormalizedLogSnapshot => ({
  entries,
  nextCursor: entries.at(-1)?.id ?? 0,
  latestId: entries.at(-1)?.id ?? 0,
  oldestId: entries[0]?.id ?? 0,
  until: entries.at(-1)?.id ?? 0,
  capacity: 512,
  modules: ["gateway"],
  hasMore: false,
  dropped: false,
  reset: false,
  recordedLevel: "INFO",
  ...overrides,
})

describe("日志增量与显示边界", () => {
  beforeEach(() => vi.clearAllMocks())

  it("分批读取固定上界，而非追赶持续增长的日志", async () => {
    vi.mocked(readLogs)
      .mockResolvedValueOnce(
        snapshot([entry(1)], { nextCursor: 1, latestId: 2, until: 2, hasMore: true }),
      )
      .mockResolvedValueOnce(snapshot([entry(2)], { latestId: 3, until: 2 }))
    const result = await readLogRange(0)
    expect(result.entries.map((item) => item.id)).toEqual([1, 2])
    expect(readLogs).toHaveBeenLastCalledWith({ after: 1, until: 2, limit: 200 })
  })

  it("合并乱序和重复记录并遵守缓存容量", () => {
    const previous = mergeLogData(undefined, snapshot([entry(1), entry(2)], { capacity: 2 }))
    const result = mergeLogData(
      previous,
      snapshot([entry(3), entry(2)], { nextCursor: 3, latestId: 3, capacity: 2 }),
    )
    expect(result.entries.map((item) => item.id)).toEqual([2, 3])
  })

  it("清空显示后，旧的在途响应不会重新显示已清空记录", () => {
    const previous = {
      ...mergeLogData(undefined, snapshot([entry(1), entry(2)])),
      entries: [],
      cursor: 2,
      clearAfter: 2,
    }
    const result = mergeLogData(previous, snapshot([entry(1), entry(2)]))
    expect(result.entries).toEqual([])
    expect(result.cursor).toBe(2)
  })

  it("新运行复位游标和清空标记", () => {
    const previous = { ...mergeLogData(undefined, snapshot([entry(99)])), clearAfter: 99 }
    const result = mergeLogData(previous, snapshot([entry(1)], { reset: true }))
    expect(result.entries.map((item) => item.id)).toEqual([1])
    expect(result.clearAfter).toBe(0)
  })

  it("按级别、模块和诊断字段组合筛选", () => {
    const entries = [entry(1), entry(2, "ERROR"), { ...entry(3), module: "storage" }]
    expect(filterLogs(entries, ["ERROR"], "gateway", "OK").map((item) => item.id)).toEqual([2])
    expect(filterLogs(entries, [], "", "")).toEqual([])
  })

  it("拒绝无法推进的游标及无效数据", async () => {
    vi.mocked(readLogs).mockResolvedValueOnce(
      snapshot([], { latestId: 1, until: 1, hasMore: true }),
    )
    await expect(readLogRange(0)).rejects.toThrow("invalid_cursor")
    vi.mocked(readLogs).mockResolvedValueOnce(snapshot([entry(1, "PRIVATE")]))
    await expect(readLogRange(0)).rejects.toThrow("invalid_snapshot")
  })

  it("取消后不继续请求下一批", async () => {
    const controller = new AbortController()
    controller.abort()
    await expect(readLogRange(0, controller.signal)).rejects.toThrow("Aborted")
    expect(readLogs).not.toHaveBeenCalled()
  })
})
