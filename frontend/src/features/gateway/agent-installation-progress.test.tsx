import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import type { AgentInstallation } from "../../services/agent-installation"
import { useUIStore } from "../../stores/ui-store"
import { AgentInstallationProgress } from "./agent-installation-progress"

const item: AgentInstallation = {
  id: "claude-code",
  detection: { state: "not_installed", path: "", version: "", errorCode: "" },
  installState: "installing",
  errorCode: "",
  officialURL: "",
  startedAt: "2026-10-12T04:00:00Z",
  finishedAt: "",
  progress: {
    phase: "downloading",
    percent: 25,
    downloadedBytes: 25 * 1024 * 1024,
    totalBytes: 100 * 1024 * 1024,
  },
}

function mount(value: AgentInstallation) {
  const client = new QueryClient()
  return render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <AgentInstallationProgress item={value} label="Claude Code CLI" starting={false} />
      </QueryClientProvider>
    </I18nProvider>,
  )
}
beforeEach(() => {
  useUIStore.setState({ language: "zh" })
  vi.useFakeTimers()
  vi.setSystemTime(new Date("2026-10-12T04:00:30Z"))
})
afterEach(() => {
  vi.useRealTimers()
})

it("显示报告的百分比、当前下载量和持续更新的已用时间", () => {
  mount(item)
  expect(screen.getByRole("progressbar", { name: "Claude Code CLI 安装进度" })).toHaveAttribute(
    "aria-valuenow",
    "25",
  )
  expect(screen.getByText("25%")).toBeInTheDocument()
  expect(screen.getByText("当前下载 25.0 MB / 100.0 MB")).toBeInTheDocument()
  expect(screen.getByText("已用 0:30")).toBeInTheDocument()
  act(() => vi.advanceTimersByTime(2000))
  expect(screen.getByText("已用 0:32")).toBeInTheDocument()
})

it("安装器未提供数值时显示不确定进度，不把下载完成当作安装完成", () => {
  const { rerender } = mount({
    ...item,
    progress: {
      phase: "installing",
      percent: null,
      downloadedBytes: 0,
      totalBytes: 0,
    },
  })
  expect(screen.getByRole("progressbar")).not.toHaveAttribute("aria-valuenow")
  expect(screen.getByText("正在安装…")).toBeInTheDocument()
  expect(screen.getByText("等待安装器反馈")).toBeInTheDocument()
  rerender(
    <I18nProvider>
      <AgentInstallationProgress
        item={{ ...item, installState: "verifying", progress: null }}
        label="Claude Code CLI"
        starting={false}
      />
    </I18nProvider>,
  )
  expect(screen.getByRole("progressbar")).not.toHaveAttribute("aria-valuenow")
  expect(screen.getByText("核对安装结果…")).toBeInTheDocument()
})

it("安装结束后移除进度并停止计时器", () => {
  const { unmount } = mount({ ...item, installState: "completed", progress: null })
  expect(screen.queryByRole("progressbar")).not.toBeInTheDocument()
  unmount()
  expect(vi.getTimerCount()).toBe(0)
})
