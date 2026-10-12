import { useCallback, useLayoutEffect, useRef, useState } from "react"
import type { LogEntry } from "../../services/logs"

const bottomThreshold = 24

export function useLogFollow(entries: LogEntry[], cursor: number) {
  const viewportRef = useRef<HTMLElement>(null)
  const [following, setFollowing] = useState(true)
  const [seenCursor, setSeenCursor] = useState(cursor)
  const anchor = useRef<{ id: string; offset: number } | null>(null)

  const rememberAnchor = useCallback(() => {
    const viewport = viewportRef.current
    if (!viewport) return
    const top = viewport.getBoundingClientRect().top
    const first = [...viewport.querySelectorAll<HTMLElement>("[data-log-id]")].find(
      (item) => item.getBoundingClientRect().bottom > top,
    )
    anchor.current = first
      ? { id: first.dataset.logId ?? "", offset: first.getBoundingClientRect().top - top }
      : null
  }, [])

  useLayoutEffect(() => {
    const viewport = viewportRef.current
    if (!viewport) return
    if (following) {
      viewport.scrollTop = viewport.scrollHeight
    } else if (anchor.current) {
      const item = [...viewport.querySelectorAll<HTMLElement>("[data-log-id]")].find(
        (row) => row.dataset.logId === anchor.current?.id,
      )
      if (item)
        viewport.scrollTop +=
          item.getBoundingClientRect().top -
          viewport.getBoundingClientRect().top -
          anchor.current.offset
    }
    rememberAnchor()
  }, [entries, cursor, following, rememberAnchor])

  function onScroll() {
    const viewport = viewportRef.current
    if (!viewport) return
    const atBottom =
      viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight <= bottomThreshold
    setFollowing(atBottom)
    if (following || atBottom) setSeenCursor(cursor)
    rememberAnchor()
  }

  return {
    viewportRef,
    following,
    onScroll,
    jump: () => setFollowing(true),
    unread: following ? 0 : Math.max(0, cursor - seenCursor),
  }
}
