import { act, fireEvent, render, screen } from "@testing-library/react"
import { StrictMode } from "react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { ScrollRegion } from "./scroll-region"

const observers: { resize: () => void; disconnected: boolean }[] = []

function resizeContent() {
  act(() => observers.filter((observer) => !observer.disconnected).forEach((item) => item.resize()))
}

function setSizes(viewport: HTMLElement, height: number, contentHeight: number) {
  Object.defineProperties(viewport, {
    clientHeight: { configurable: true, value: height },
    scrollHeight: { configurable: true, value: contentHeight },
  })
  Object.defineProperty(viewport.nextElementSibling, "clientHeight", {
    configurable: true,
    value: height,
  })
}

function scrollTo(viewport: HTMLElement, top: number) {
  viewport.scrollTop = top
  fireEvent.scroll(viewport)
}

function page(key: string) {
  return (
    <StrictMode>
      <ScrollRegion as="main" pageKey={key}>
        {key}
      </ScrollRegion>
    </StrictMode>
  )
}

describe("页面滚动位置恢复", () => {
  beforeEach(() => {
    observers.length = 0
    vi.stubGlobal(
      "ResizeObserver",
      class {
        item: (typeof observers)[number]
        constructor(callback: () => void) {
          this.item = { resize: callback, disconnected: false }
          observers.push(this.item)
        }
        observe() {}
        disconnect() {
          this.item.disconnected = true
        }
      },
    )
  })
  afterEach(() => vi.unstubAllGlobals())

  it("首次访问从顶部开始，来回切换恢复各自位置并同步滑块", () => {
    const { rerender } = render(page("/providers"))
    const viewport = screen.getByRole("main")
    setSizes(viewport, 100, 1000)
    scrollTo(viewport, 450)
    rerender(page("/settings"))
    expect(viewport.scrollTop).toBe(0)
    scrollTo(viewport, 180)
    rerender(page("/providers"))
    expect(viewport.scrollTop).toBe(450)
    const thumb = viewport.nextElementSibling?.firstElementChild as HTMLElement
    expect(thumb.style.transform).toBe("translateY(38px)")
    rerender(page("/settings"))
    expect(viewport.scrollTop).toBe(180)
  })

  it("异步加载的临时短内容不覆盖记录，高度恢复后返回原位置", () => {
    const { rerender } = render(page("/quota"))
    const viewport = screen.getByRole("main")
    setSizes(viewport, 100, 1000)
    scrollTo(viewport, 600)
    rerender(page("/about"))
    setSizes(viewport, 100, 120)
    rerender(page("/quota"))
    expect(viewport.scrollTop).toBe(20)
    fireEvent.scroll(viewport)
    setSizes(viewport, 100, 350)
    resizeContent()
    expect(viewport.scrollTop).toBe(250)
    fireEvent.scroll(viewport)
    rerender(page("/about"))
    rerender(page("/quota"))
    setSizes(viewport, 100, 1000)
    resizeContent()
    expect(viewport.scrollTop).toBe(600)
  })

  it.each(["wheel", "touchStart", "pointerDown", "keyDown"] as const)(
    "等待期间用户通过 %s 操作后，不再被后续加载拉回",
    (event) => {
      const { rerender } = render(page("/quota"))
      const viewport = screen.getByRole("main")
      setSizes(viewport, 100, 1000)
      scrollTo(viewport, 600)
      rerender(page("/about"))
      setSizes(viewport, 100, 300)
      rerender(page("/quota"))
      fireEvent[event](viewport, { key: "PageUp" })
      scrollTo(viewport, 80)
      setSizes(viewport, 100, 1000)
      resizeContent()
      expect(viewport.scrollTop).toBe(80)
      rerender(page("/about"))
      rerender(page("/quota"))
      expect(viewport.scrollTop).toBe(80)
    },
  )

  it("离开页面后过期的尺寸回调不会覆盖新页面位置", () => {
    const { rerender } = render(page("/quota"))
    const viewport = screen.getByRole("main")
    setSizes(viewport, 100, 1000)
    scrollTo(viewport, 600)
    rerender(page("/about"))
    setSizes(viewport, 100, 300)
    rerender(page("/quota"))
    const pendingObserver = observers.at(-1)!
    rerender(page("/settings"))
    scrollTo(viewport, 120)
    setSizes(viewport, 100, 1000)
    pendingObserver.resize()
    expect(viewport.scrollTop).toBe(120)
  })
})
