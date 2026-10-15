import { act, cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { AgentConfigSuccessToast } from "./agent-config-success-toast"

function mount() {
  return render(
    <I18nProvider>
      <div className="relative">
        <button>原操作</button>
        <AgentConfigSuccessToast />
      </div>
    </I18nProvider>,
  )
}
const advance = async (duration: number) => {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(duration)
  })
}
beforeEach(() => {
  vi.useFakeTimers()
  useUIStore.setState({ language: "zh" })
})
afterEach(() => {
  cleanup()
  vi.useRealTimers()
})

it("announces the complete success message politely, keeps focus, and dismisses after five seconds", async () => {
  mount()
  act(() => screen.getByRole("button", { name: "原操作" }).focus())
  await advance(100)
  expect(screen.getByRole("status")).toHaveAttribute("aria-live", "polite")
  expect(screen.getByRole("status")).toHaveTextContent("配置已保存，原配置已加密备份。")
  expect(screen.getByRole("button", { name: "原操作" })).toHaveFocus()
  await advance(4800)
  expect(screen.getByRole("button", { name: "关闭保存成功提示" })).toBeInTheDocument()
  await advance(200)
  expect(screen.queryByRole("button", { name: "关闭保存成功提示" })).not.toBeInTheDocument()
})
it("pauses dismissal while the pointer is over the notice", async () => {
  mount()
  await advance(1000)
  const viewport = screen.getByRole("region", { name: "配置保存通知" })
  fireEvent.pointerMove(viewport)
  await advance(6000)
  expect(screen.getByRole("button", { name: "关闭保存成功提示" })).toBeInTheDocument()
  fireEvent.pointerLeave(viewport)
  await advance(4100)
  expect(screen.queryByRole("button", { name: "关闭保存成功提示" })).not.toBeInTheDocument()
})
it("pauses dismissal while a keyboard user focuses the close control", async () => {
  mount()
  await advance(1000)
  act(() => screen.getByRole("button", { name: "关闭保存成功提示" }).focus())
  await advance(6000)
  expect(screen.getByRole("button", { name: "关闭保存成功提示" })).toHaveFocus()
  act(() => screen.getByRole("button", { name: "原操作" }).focus())
  await advance(4100)
  expect(screen.queryByRole("button", { name: "关闭保存成功提示" })).not.toBeInTheDocument()
})
