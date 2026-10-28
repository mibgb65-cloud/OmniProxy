import { useCallback, useLayoutEffect, useRef, type RefObject } from "react"

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

/**
 * FLIP 布局过渡（零依赖）。
 *
 * 用法：在改变布局的状态更新**之前**调用 `capture()`，布局提交后由内部的布局副作用播放动画。
 * 每个参与过渡的元素用 `data-flip-id` 标识，动画只做 `translate + scale` 补间，
 * 因此不需要知道具体的网格行列（展开、压缩重排、反向收起共用同一套逻辑）。
 */
export function useFlipReflow(
  containerRef: RefObject<HTMLElement | null>,
  { duration = DEFAULT_DURATION, easing = DEFAULT_EASING } = {},
): () => void {
  const previousRects = useRef(new Map<string, DOMRect>())
  const pending = useRef(false)

  const capture = useCallback(() => {
    const container = containerRef.current
    if (!container) return
    const rects = new Map<string, DOMRect>()
    for (const element of container.querySelectorAll<HTMLElement>(FLIP_SELECTOR)) {
      const id = element.dataset.flipId
      if (id) rects.set(id, element.getBoundingClientRect())
    }
    previousRects.current = rects
    pending.current = true
  }, [containerRef])

  useLayoutEffect(() => {
    const container = containerRef.current
    if (!pending.current || !container) return
    pending.current = false
    const before = previousRects.current
    previousRects.current = new Map()
    if (prefersReducedMotion()) return

    for (const element of container.querySelectorAll<HTMLElement>(FLIP_SELECTOR)) {
      const id = element.dataset.flipId
      const first = id ? before.get(id) : undefined
      // jsdom 没有布局也没有 Element.animate：矩形为 0 或缺少 API 时保持静态。
      if (!first || typeof element.animate !== "function") continue

      const last = element.getBoundingClientRect()
      if (last.width === 0 || last.height === 0) continue

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

      const animation = element.animate(
        [
          {
            transformOrigin: "top left",
            transform: `translate(${dx}px, ${dy}px) scale(${scaleX}, ${scaleY})`,
          },
          { transformOrigin: "top left", transform: "none" },
        ],
        { duration, easing },
      )
      // 尺寸差异大时缩放会压扁文字，飞行期间淡化内部内容，落位后再交回真实内容。
      element.dataset.flipFlying = "true"
      const settle = () => {
        delete element.dataset.flipFlying
      }
      animation.finished.then(settle, settle)
    }
  })

  return capture
}
