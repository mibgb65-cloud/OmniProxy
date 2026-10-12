import { selectOption } from "../../test/select-option"
import { Clipboard } from "@wailsio/runtime"
import { act, fireEvent, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import {
  exportLogs,
  getLogHead,
  openLogDirectory,
  readLogs,
  type LogEntry,
} from "../../services/logs"
import { useUIStore } from "../../stores/ui-store"
import { makeEntry, mockLogReader, renderLogs } from "./logs-test-helpers"

const runtime = vi.hoisted(() => ({
  handlers: new Map<string, (event: { data: { latestId: number } }) => void>(),
}))
vi.mock("../../services/logs", () => ({
  readLogs: vi.fn(),
  getLogHead: vi.fn(),
  exportLogs: vi.fn(),
  openLogDirectory: vi.fn(),
}))
vi.mock("@wailsio/runtime", () => ({
  Clipboard: { SetText: vi.fn() },
  Events: {
    On: (name: string, callback: (event: { data: { latestId: number } }) => void) => {
      runtime.handlers.set(name, callback)
      return () => runtime.handlers.delete(name)
    },
  },
}))

let entries: LogEntry[]
function notify() {
  act(() => runtime.handlers.get("logs:updated")?.({ data: { latestId: entries.at(-1)?.id ?? 0 } }))
}

describe("实时日志页面", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    runtime.handlers.clear()
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    entries = [
      makeEntry(1, "INFO", "app", "application ready"),
      makeEntry(2, "WARN", "gateway", "provider warning"),
      makeEntry(3, "ERROR", "gateway", "request failed"),
      makeEntry(4, "DEBUG", "routing", "routing debug"),
    ]
    mockLogReader(() => entries)
    vi.mocked(getLogHead).mockImplementation(async () => entries.at(-1)?.id ?? 0)
    vi.mocked(Clipboard.SetText).mockResolvedValue()
    vi.mocked(exportLogs).mockResolvedValue({ count: 2, missing: 0, cancelled: false })
    vi.mocked(openLogDirectory).mockResolvedValue()
  })

  it("默认隐藏 Debug，可按模块和关键词筛选，并展开诊断字段", async () => {
    const user = userEvent.setup()
    renderLogs()
    await screen.findByText("application ready")
    expect(screen.queryByText("routing debug")).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "DEBUG" }))
    expect(screen.getByText("routing debug")).toBeInTheDocument()
    await selectOption(user, screen.getByRole("combobox", { name: "筛选模块" }), "gateway")
    expect(screen.queryByText("application ready")).not.toBeInTheDocument()
    await user.type(screen.getByRole("searchbox", { name: "搜索日志" }), "failed")
    await waitFor(() => expect(screen.queryByText("provider warning")).not.toBeInTheDocument())
    await user.click(screen.getByText("request failed"))
    expect(screen.getByText("deepseek-flash")).toBeVisible()
  })

  it("复制完整安全记录，复制失败可重试", async () => {
    const user = userEvent.setup()
    renderLogs()
    await screen.findByText("application ready")
    vi.mocked(Clipboard.SetText).mockRejectedValueOnce(new Error("private clipboard error"))
    await user.click(screen.getByRole("button", { name: "复制日志 1" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("复制失败")
    expect(screen.queryByText(/private clipboard/)).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "复制日志 1" }))
    expect(JSON.parse(vi.mocked(Clipboard.SetText).mock.calls.at(-1)![0])).toMatchObject({
      id: 1,
      message: "application ready",
    })
  })

  it("暂停保留当前视图，提示新增数量并在继续时补齐", async () => {
    const user = userEvent.setup()
    renderLogs()
    await screen.findByText("application ready")
    await user.click(screen.getByRole("button", { name: "暂停更新" }))
    entries.push(makeEntry(5, "INFO", "gateway", "after pause"))
    notify()
    expect(screen.queryByText("after pause")).not.toBeInTheDocument()
    await screen.findByRole("button", { name: "新增 1 条 · 回到最新" })
    await user.click(screen.getByRole("button", { name: "继续更新" }))
    expect(await screen.findByText("after pause")).toBeInTheDocument()
  })

  it("向上翻阅时不自动跳动，并可以回到最新", async () => {
    const user = userEvent.setup()
    renderLogs()
    await screen.findByText("application ready")
    const viewport = screen.getByRole("region", { name: "运行日志列表" })
    Object.defineProperties(viewport, {
      clientHeight: { configurable: true, value: 100 },
      scrollHeight: { configurable: true, value: 400 },
    })
    viewport.scrollTop = 0
    fireEvent.scroll(viewport)
    entries.push(makeEntry(5))
    notify()
    await screen.findByText("event 5")
    expect(viewport.scrollTop).toBe(0)
    await user.click(screen.getByRole("button", { name: "新增 1 条 · 回到最新" }))
    await waitFor(() => expect(viewport.scrollTop).toBeGreaterThanOrEqual(300))
  })

  it("清空显示保留服务端记录，并只显示后续新增日志", async () => {
    const user = userEvent.setup()
    renderLogs()
    await screen.findByText("application ready")
    await user.click(screen.getByRole("button", { name: "清空显示" }))
    await screen.findByText("暂无日志，新的运行记录会显示在这里。")
    expect(entries).toHaveLength(4)
    entries.push(makeEntry(5, "INFO", "app", "fresh record"))
    notify()
    await screen.findByText("fresh record")
    expect(screen.queryByText("application ready")).not.toBeInTheDocument()
  })

  it("导出当前筛选 ID 与打开目录，取消导出不显示成功", async () => {
    const user = userEvent.setup()
    renderLogs()
    await screen.findByText("application ready")
    await selectOption(user, screen.getByRole("combobox", { name: "筛选模块" }), "gateway")
    vi.mocked(exportLogs).mockResolvedValueOnce({ count: 0, missing: 0, cancelled: true })
    await user.click(screen.getByRole("button", { name: "导出筛选结果" }))
    expect(exportLogs).toHaveBeenCalledWith([2, 3], "zh")
    expect(screen.queryByText(/已导出/)).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "打开日志目录" }))
    expect(openLogDirectory).toHaveBeenCalledTimes(1)
  })

  it("安全显示读取失败并允许重试", async () => {
    const user = userEvent.setup()
    vi.mocked(readLogs).mockRejectedValueOnce(new Error("private failure"))
    renderLogs()
    expect(await screen.findByRole("alert")).toHaveTextContent("无法读取日志")
    expect(screen.queryByText(/private failure/)).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "重试" }))
    await screen.findByText("application ready")
  })

  it("切换英文并在离开页面时解除订阅", async () => {
    const result = renderLogs()
    await screen.findByText("application ready")
    act(() => useUIStore.setState({ language: "en" }))
    expect(await screen.findByRole("button", { name: "Export filtered logs" })).toBeInTheDocument()
    expect(runtime.handlers.size).toBe(1)
    result.unmount()
    expect(runtime.handlers.size).toBe(0)
  })
})
