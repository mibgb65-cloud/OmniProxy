import { useRef, useState } from "react"
import { act, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { useFlipReflow } from "./use-flip-reflow"

function Fixture() {
  const containerRef = useRef<HTMLDivElement>(null)
  const capture = useFlipReflow(containerRef)
  const [expanded, setExpanded] = useState(false)
  return (
    <>
      <button
        onClick={() => {
          capture()
          setExpanded((value) => !value)
        }}
      >
        toggle
      </button>
      <div ref={containerRef} data-expanded={expanded}>
        <div data-flip-id="provider">
          <div>content</div>
        </div>
      </div>
    </>
  )
}

type RecordedAnimation = {
  element: HTMLElement
  frames: Keyframe[] | PropertyIndexedKeyframes | null
  options?: number | KeyframeAnimationOptions
  animation: Animation
  finish: () => void
}
let records: RecordedAnimation[]
let measured: number
const originalAnimate = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "animate")

beforeEach(() => {
  records = []
  measured = 0
  vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(function (
    this: HTMLElement,
  ) {
    measured++
    if (this.classList.contains("bento-motion-surface")) return new DOMRect(100, 50, 600, 400)
    const expanded = this.parentElement?.dataset.expanded === "true"
    return new DOMRect(
      expanded ? 0 : 300,
      expanded ? 0 : 150,
      expanded ? 800 : 400,
      expanded ? 500 : 320,
    )
  })
  vi.stubGlobal(
    "matchMedia",
    vi.fn(() => ({ matches: false })),
  )
  vi.stubGlobal("Animation", class {})
  Object.defineProperty(HTMLElement.prototype, "animate", {
    configurable: true,
    writable: true,
    value: vi.fn(),
  })
  vi.spyOn(HTMLElement.prototype, "animate").mockImplementation(function (
    this: HTMLElement,
    frames,
    options,
  ) {
    let finish = () => {}
    const finished = new Promise<Animation>((resolve) => {
      finish = () => resolve(animation)
    })
    const animation = {
      cancel: vi.fn(),
      finished,
    } as unknown as Animation
    records.push({ element: this, frames, options, animation, finish })
    return animation
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  if (originalAnimate) Object.defineProperty(HTMLElement.prototype, "animate", originalAnimate)
  else Reflect.deleteProperty(HTMLElement.prototype, "animate")
})

describe("useFlipReflow", () => {
  it("keeps 600ms geometry and inverse content animations without fading the panel", () => {
    render(<Fixture />)
    fireEvent.click(screen.getByText("toggle"))
    expect(records).toHaveLength(3)
    expect(records[0].frames).toEqual([
      { transform: "translate(300px, 150px) scale(0.5, 0.64)" },
      { transform: "none" },
    ])
    expect(records[0].options).toEqual({
      duration: 600,
      easing: "cubic-bezier(0.22, 1, 0.36, 1)",
      fill: "both",
    })
    const content = records[2].frames as Keyframe[]
    expect(content[0].transform).toBe("scale(2, 1.5625)")
    expect(content.at(-1)?.transform).toBe("scale(1, 1)")
    const outgoing = records[1].frames as Keyframe[]
    expect(
      content.every(
        (frame, index) => Number(frame.opacity) + Number(outgoing[index].opacity) === 1,
      ),
    ).toBe(true)
    expect(document.querySelector("[data-flip-id]")).toHaveAttribute("data-flip-flying", "true")
    expect(records[0].element).toHaveAttribute("aria-hidden", "true")
  })

  it("samples the displayed rect before cancelling and ignores stale completion callbacks", async () => {
    render(<Fixture />)
    fireEvent.click(screen.getByText("toggle"))
    const old = [...records]
    const reads = measured
    vi.mocked(old[0].animation.cancel).mockImplementation(() => {
      expect(measured).toBeGreaterThan(reads)
    })
    fireEvent.click(screen.getByText("toggle"))
    expect(old.every(({ animation }) => vi.mocked(animation.cancel).mock.calls.length === 1)).toBe(
      true,
    )
    const current = records[3]
    const target = document.querySelector("[data-flip-id]")
    await act(async () => old[0].finish())
    expect(target).toHaveAttribute("data-flip-flying", "true")
    expect(current.animation.cancel).not.toHaveBeenCalled()
    await act(async () => current.finish())
    expect(target).not.toHaveAttribute("data-flip-flying")
    expect(current.animation.cancel).toHaveBeenCalledOnce()
  })

  it("cancels all live animations when unmounted", () => {
    const { unmount } = render(<Fixture />)
    fireEvent.click(screen.getByText("toggle"))
    unmount()
    expect(
      records.every(({ animation }) => vi.mocked(animation.cancel).mock.calls.length === 1),
    ).toBe(true)
    expect(records[0].element.dataset.flipFlying).toBeUndefined()
    expect(document.querySelector(".bento-motion-surface")).not.toBeInTheDocument()
  })

  it("skips animations for reduced motion", () => {
    vi.stubGlobal(
      "matchMedia",
      vi.fn(() => ({ matches: true })),
    )
    render(<Fixture />)
    fireEvent.click(screen.getByText("toggle"))
    expect(records).toHaveLength(0)
  })

  it.each(["resize", "wheel", "touchstart"])("cleans up stale geometry on %s", (event) => {
    render(<Fixture />)
    fireEvent.click(screen.getByText("toggle"))
    fireEvent(window, new Event(event))
    expect(document.querySelector(".bento-motion-surface")).not.toBeInTheDocument()
    expect(document.querySelector("[data-flip-id]")).not.toHaveAttribute("data-flip-flying")
    expect(
      records.every(({ animation }) => vi.mocked(animation.cancel).mock.calls.length === 1),
    ).toBe(true)
  })

  it("skips zero-size geometry and works without the Web Animations API", () => {
    vi.mocked(HTMLElement.prototype.getBoundingClientRect).mockReturnValue(new DOMRect())
    const { unmount } = render(<Fixture />)
    fireEvent.click(screen.getByText("toggle"))
    expect(records).toHaveLength(0)
    unmount()
    Reflect.deleteProperty(HTMLElement.prototype, "animate")
    render(<Fixture />)
    expect(() => fireEvent.click(screen.getByText("toggle"))).not.toThrow()
  })
})
