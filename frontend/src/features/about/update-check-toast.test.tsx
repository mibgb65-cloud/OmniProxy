import { act, cleanup, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { UpdateCheckToast } from "./update-check-toast"

function mount(container?: HTMLElement) {
  return render(
    <I18nProvider>
      <button>原操作</button>
      <UpdateCheckToast container={container} />
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

it("礼貌播报最新版本，不抢焦点，五秒后自动消失", async () => {
  mount()
  act(() => screen.getByRole("button", { name: "原操作" }).focus())
  await advance(100)
  expect(screen.getByRole("status")).toHaveAttribute("aria-live", "polite")
  expect(screen.getByRole("status")).toHaveTextContent("已是最新版本。")
  expect(screen.getByRole("button", { name: "原操作" })).toHaveFocus()
  await advance(4800)
  expect(screen.getByRole("button", { name: "关闭更新提示" })).toBeInTheDocument()
  await advance(200)
  expect(screen.queryByRole("button", { name: "关闭更新提示" })).not.toBeInTheDocument()
})

it("悬停时暂停计时，移开后自动消失", async () => {
  mount()
  await advance(1000)
  const viewport = screen.getByRole("region", { name: "更新检查通知" })
  fireEvent.pointerMove(viewport)
  await advance(6000)
  expect(screen.getByRole("button", { name: "关闭更新提示" })).toBeInTheDocument()
  fireEvent.pointerLeave(viewport)
  await advance(4100)
  expect(screen.queryByRole("button", { name: "关闭更新提示" })).not.toBeInTheDocument()
})

it("关闭按钮获得键盘焦点时暂停计时，并可手动关闭", async () => {
  mount()
  await advance(1000)
  const close = screen.getByRole("button", { name: "关闭更新提示" })
  act(() => close.focus())
  await advance(6000)
  expect(close).toHaveFocus()
  fireEvent.click(close)
  expect(screen.queryByRole("button", { name: "关闭更新提示" })).not.toBeInTheDocument()
})

it("提示挂载到提供的非滚动容器，组件卸载后清理", async () => {
  const host = document.createElement("div")
  document.body.append(host)
  try {
    const view = mount(host)
    expect(host.querySelector("[data-slot='update-check-toast']")).toBeInTheDocument()
    view.unmount()
    expect(host.childElementCount).toBe(0)
  } finally {
    host.remove()
  }
})

it("英文提示与关闭按钮翻译完整", async () => {
  useUIStore.setState({ language: "en" })
  mount()
  await advance(100)
  expect(screen.getByRole("status")).toHaveTextContent("You are running the latest version.")
  expect(screen.getByRole("button", { name: "Dismiss update notification" })).toBeInTheDocument()
})
