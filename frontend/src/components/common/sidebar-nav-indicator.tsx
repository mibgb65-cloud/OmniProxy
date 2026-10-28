import { useCallback, useLayoutEffect, useRef, useState } from "react"

type IndicatorBox = { top: number; height: number }

/**
 * 侧栏导航的选中背景（滑动药丸）。
 *
 * 位置与高度都取自当前选中项的真实布局，因此相邻切换、跨项跳转、侧栏收展与全局字号变化
 * 共用同一套测量逻辑，不需要为每种情况单独写偏移表。选中判定直接观察路由写在链接上的
 * `data-active`，不复制路由匹配规则。
 *
 * 列表元素通过回调 ref 以状态传入，而不是引用对象：这样首次挂载时也能等元素真正就位后
 * 再测量，不依赖兄弟节点的副作用执行顺序。
 */
export function SidebarNavIndicator({ list }: { list: HTMLElement | null }) {
  const [box, setBox] = useState<IndicatorBox | null>(null)
  // 首次测量完成后才允许过渡，避免挂载时从容器顶部滑下来的误触发动画。
  const [measured, setMeasured] = useState(false)
  const pendingFrame = useRef(0)

  const measure = useCallback(() => {
    const active = list?.querySelector<HTMLElement>('[data-active="true"]')
    setBox((previous) => {
      if (!active) return null
      const next = { top: active.offsetTop, height: active.offsetHeight }
      // 值没变时复用旧对象：观察器与状态更新不会互相触发。
      if (previous && previous.top === next.top && previous.height === next.height) return previous
      return next
    })
  }, [list])

  useLayoutEffect(() => {
    // 位置只能从提交后的真实布局读出，再回写成状态驱动渲染；这是布局测量的正当例外，
    // 不存在渲染期副作用，也不会因此产生额外渲染循环（值未变时复用旧对象）。
    // eslint-disable-next-line react-hooks/set-state-in-effect
    measure()
  }, [measure])

  // data-active 的变化即「选中项换了」；字号缩放会改变 rem 行高，容器尺寸也要跟着重量。
  useLayoutEffect(() => {
    if (!list) return
    const selection = new MutationObserver(measure)
    selection.observe(list, {
      attributes: true,
      attributeFilter: ["data-active"],
      subtree: true,
    })
    const resize = new ResizeObserver(measure)
    resize.observe(list)
    return () => {
      selection.disconnect()
      resize.disconnect()
    }
  }, [list, measure])

  useLayoutEffect(() => {
    if (!box || measured) return
    pendingFrame.current = requestAnimationFrame(() => setMeasured(true))
    return () => cancelAnimationFrame(pendingFrame.current)
  }, [box, measured])

  if (!box) return null

  return (
    <div
      aria-hidden="true"
      className="sidebar-nav-indicator"
      data-measured={measured}
      style={{ transform: `translateY(${box.top}px)`, height: `${box.height}px` }}
    />
  )
}
