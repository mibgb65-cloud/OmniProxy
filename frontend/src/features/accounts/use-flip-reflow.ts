import { useCallback, useEffect, useLayoutEffect, useRef, type RefObject } from "react"
import { animateFlipSnapshot, takeFlipSnapshot, type FlipSnapshot } from "./flip-snapshot"

const FLIP_SELECTOR = "[data-flip-id]"
const DEFAULT_DURATION = 600
const DEFAULT_EASING = "cubic-bezier(0.22, 1, 0.36, 1)"

/** 系统的「减少动画」偏好：此时直接切换布局，不做 FLIP。 */
function prefersReducedMotion(): boolean {
  return (
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches
  )
}

/** 零依赖、可打断的 600ms FLIP。必须在更新布局状态前调用 capture。 */
export function useFlipReflow(
  containerRef: RefObject<HTMLElement | null>,
  { duration = DEFAULT_DURATION, easing = DEFAULT_EASING } = {},
): () => void {
  const previousRects = useRef(new Map<string, FlipSnapshot>())
  const pending = useRef(false)
  const running = useRef(new Map<HTMLElement, ReturnType<typeof animateFlipSnapshot>>())
  const cancelAnimations = useCallback(() => {
    for (const [element, { animations, surface }] of running.current) {
      for (const animation of animations) animation.cancel()
      surface.remove()
      delete element.dataset.flipFlying
    }
    running.current.clear()
  }, [])

  useEffect(() => {
    // 窗口尺寸或用户滚动改变了目标坐标时直接交回真实布局，不保留过期的固定过渡层。
    window.addEventListener("resize", cancelAnimations)
    window.addEventListener("wheel", cancelAnimations, { passive: true })
    window.addEventListener("touchstart", cancelAnimations, { passive: true })
    return () => {
      window.removeEventListener("resize", cancelAnimations)
      window.removeEventListener("wheel", cancelAnimations)
      window.removeEventListener("touchstart", cancelAnimations)
      cancelAnimations()
    }
  }, [cancelAnimations])

  const capture = useCallback(() => {
    const container = containerRef.current
    if (!container) return
    const rects = new Map<string, FlipSnapshot>()
    for (const element of container.querySelectorAll<HTMLElement>(FLIP_SELECTOR)) {
      const id = element.dataset.flipId
      if (id) rects.set(id, takeFlipSnapshot(running.current.get(element)?.surface ?? element))
    }
    // 先取当前呈现位置再取消，下一次动画从屏幕上的位置继续，而非跳回上一个终点。
    cancelAnimations()
    previousRects.current = rects
    pending.current = true
  }, [containerRef, cancelAnimations])

  useLayoutEffect(() => {
    const container = containerRef.current
    if (!pending.current || !container) return
    pending.current = false
    const before = previousRects.current
    previousRects.current = new Map()
    if (prefersReducedMotion()) return

    // 先读取所有终点再写动画，避免前一个动画影响后续布局测量。
    const targets = Array.from(
      container.querySelectorAll<HTMLElement>(FLIP_SELECTOR),
      (element) => ({
        element,
        last: element.getBoundingClientRect(),
      }),
    )
    for (const { element, last } of targets) {
      const id = element.dataset.flipId
      const snapshot = id ? before.get(id) : undefined
      const first = snapshot?.rect
      // jsdom 没有布局也没有 Element.animate：矩形为 0 或缺少 API 时保持静态。
      if (!snapshot || !first || typeof element.animate !== "function") continue

      if (first.width <= 0 || first.height <= 0 || last.width <= 0 || last.height <= 0) continue

      const dx = first.left - last.left
      const dy = first.top - last.top
      const scaleX = first.width / last.width
      const scaleY = first.height / last.height
      const settled =
        Math.abs(dx) < 1 &&
        Math.abs(dy) < 1 &&
        Math.abs(scaleX - 1) < 0.01 &&
        Math.abs(scaleY - 1) < 0.01
      if (settled) continue

      const options = { duration, easing, fill: "both" as const }
      const flight = animateFlipSnapshot(snapshot, element, last, options)
      element.dataset.flipFlying = "true"
      running.current.set(element, flight)
      const settle = () => {
        // 被打断的旧回调不能清理同一 DOM 上的新动画。
        if (running.current.get(element) !== flight) return
        for (const item of flight.animations) item.cancel()
        flight.surface.remove()
        running.current.delete(element)
        delete element.dataset.flipFlying
      }
      flight.animation.finished.then(settle, settle)
    }
  })

  return capture
}
