import { useCallback, useLayoutEffect, useRef } from "react"
import type { RefObject } from "react"

export function useScrollRestoration(
  viewportRef: RefObject<HTMLElement | null>,
  contentRef: RefObject<HTMLElement | null>,
  pageKey: string | undefined,
  syncThumb: () => void,
) {
  const positions = useRef(new Map<string, number>())
  const active = useRef<{ key: string; pending: number | null } | null>(null)

  useLayoutEffect(() => {
    const viewport = viewportRef.current
    const content = contentRef.current
    if (pageKey === undefined || !viewport || !content) {
      active.current = null
      return
    }

    const state: { key: string; pending: number | null } = {
      key: pageKey,
      pending: positions.current.get(pageKey) ?? 0,
    }
    active.current = state
    const restore = () => {
      if (active.current !== state || state.pending === null) return
      const maxScroll = Math.max(0, viewport.scrollHeight - viewport.clientHeight)
      viewport.scrollTop = Math.min(state.pending, maxScroll)
      syncThumb()
      // 异步页面可能先显示很短的加载态；保留目标，等内容足够高再完成恢复。
      if (maxScroll >= state.pending) state.pending = null
    }
    const observer = new ResizeObserver(restore)
    observer.observe(viewport)
    observer.observe(content)
    restore()
    return () => observer.disconnect()
  }, [pageKey, viewportRef, contentRef, syncThumb])

  const recordScroll = useCallback(() => {
    const viewport = viewportRef.current
    const state = active.current
    if (!viewport || !state || state.pending !== null) return
    positions.current.set(state.key, Math.max(0, viewport.scrollTop))
  }, [viewportRef])

  const cancelRestoration = useCallback(() => {
    const state = active.current
    if (!state || state.pending === null) return
    // 用户主动滚动／操作后以当前位置为准，后续加载不再把用户拉回旧位置。
    state.pending = null
    recordScroll()
  }, [recordScroll])

  return { recordScroll, cancelRestoration }
}
