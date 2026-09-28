import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { ScrollRegion } from "./scroll-region"

function mockScrollSize(viewport: HTMLElement, height: number, contentHeight: number) {
  Object.defineProperties(viewport, {
    clientHeight: { configurable: true, value: height },
    scrollHeight: { configurable: true, value: contentHeight },
  })
  const rail = viewport.nextElementSibling as HTMLDivElement
  Object.defineProperty(rail, "clientHeight", { configurable: true, value: height })
  return { rail, thumb: rail.firstElementChild as HTMLDivElement }
}

describe("ScrollRegion", () => {
  it("tracks each pane's scroll position independently", () => {
    render(
      <>
        <ScrollRegion as="main">Main content</ScrollRegion>
        <ScrollRegion as="nav" ariaLabel="Primary">
          Navigation
        </ScrollRegion>
      </>,
    )

    const main = screen.getByRole("main")
    const nav = screen.getByRole("navigation", { name: "Primary" })
    const mainScrollbar = mockScrollSize(main, 100, 400)
    const navScrollbar = mockScrollSize(nav, 100, 300)

    fireEvent.scroll(main)
    fireEvent.scroll(nav)
    expect(mainScrollbar.rail).toHaveAttribute("data-scrollable", "true")
    expect(mainScrollbar.thumb.style.height).toBe("25px")
    expect(navScrollbar.rail).toHaveAttribute("data-scrollable", "true")

    main.scrollTop = 150
    fireEvent.scroll(main)
    expect(mainScrollbar.thumb.style.transform).toBe("translateY(37.5px)")
    expect(nav.scrollTop).toBe(0)
    expect(navScrollbar.thumb.style.transform).toBe("translateY(0px)")
  })

  it("keeps the reserved rail interactive while the thumb is hidden", () => {
    render(<ScrollRegion as="main">Main content</ScrollRegion>)
    const viewport = screen.getByRole("main")
    const { rail, thumb } = mockScrollSize(viewport, 100, 400)
    fireEvent.scroll(viewport)

    Object.defineProperty(thumb, "offsetHeight", {
      configurable: true,
      get: () => Number.parseFloat(thumb.style.height),
    })
    rail.getBoundingClientRect = () => ({ top: 0 }) as DOMRect
    rail.setPointerCapture = vi.fn()
    rail.hasPointerCapture = vi.fn(() => true)
    rail.releasePointerCapture = vi.fn()

    fireEvent.pointerDown(rail, { button: 0, pointerId: 1, clientY: 75 })
    expect(viewport.scrollTop).toBe(250)
    expect(rail).toHaveAttribute("data-dragging", "true")
    fireEvent.pointerMove(rail, { pointerId: 1, clientY: 90 })
    expect(viewport.scrollTop).toBe(300)
    fireEvent.pointerUp(rail, { pointerId: 1 })
    expect(rail).not.toHaveAttribute("data-dragging")
    expect(rail.releasePointerCapture).toHaveBeenCalledWith(1)
  })
})
