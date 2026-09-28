import { useCallback, useLayoutEffect, useRef } from "react"
import type { PointerEvent, ReactNode } from "react"
import { cn } from "../../lib/utils"
import "./scroll-region.css"

const MIN_THUMB_HEIGHT = 24

type ScrollRegionProps = {
  as: "main" | "nav" | "section"
  children: ReactNode
  className?: string
  viewportClassName?: string
  ariaLabel?: string
}

export function ScrollRegion({
  as: Element,
  children,
  className,
  viewportClassName,
  ariaLabel,
}: ScrollRegionProps) {
  const viewportRef = useRef<HTMLElement>(null)
  const contentRef = useRef<HTMLDivElement>(null)
  const railRef = useRef<HTMLDivElement>(null)
  const thumbRef = useRef<HTMLDivElement>(null)
  const dragRef = useRef<{ pointerId: number; offset: number } | null>(null)

  const syncThumb = useCallback(() => {
    const viewport = viewportRef.current
    const rail = railRef.current
    const thumb = thumbRef.current
    if (!viewport || !rail || !thumb) return

    const maxScroll = viewport.scrollHeight - viewport.clientHeight
    const railHeight = rail.clientHeight
    const scrollable = maxScroll > 1 && railHeight > 0
    rail.dataset.scrollable = String(scrollable)
    if (!scrollable) return

    const thumbHeight = Math.min(
      railHeight,
      Math.max(MIN_THUMB_HEIGHT, (railHeight * viewport.clientHeight) / viewport.scrollHeight),
    )
    const thumbTop = (viewport.scrollTop / maxScroll) * (railHeight - thumbHeight)
    thumb.style.height = `${thumbHeight}px`
    thumb.style.transform = `translateY(${thumbTop}px)`
  }, [])

  useLayoutEffect(() => {
    const viewport = viewportRef.current
    const content = contentRef.current
    const rail = railRef.current
    if (!viewport || !content || !rail) return

    const observer = new ResizeObserver(syncThumb)
    observer.observe(viewport)
    observer.observe(content)
    observer.observe(rail)
    syncThumb()
    return () => observer.disconnect()
  }, [syncThumb])

  const scrollToPointer = (clientY: number, offset: number) => {
    const viewport = viewportRef.current
    const rail = railRef.current
    const thumb = thumbRef.current
    if (!viewport || !rail || !thumb) return

    const maxScroll = viewport.scrollHeight - viewport.clientHeight
    const travel = rail.clientHeight - thumb.offsetHeight
    if (maxScroll <= 0 || travel <= 0) return

    const position = clientY - rail.getBoundingClientRect().top - offset
    viewport.scrollTop = (Math.max(0, Math.min(position, travel)) / travel) * maxScroll
    syncThumb()
  }

  const handlePointerDown = (event: PointerEvent<HTMLDivElement>) => {
    const rail = railRef.current
    const thumb = thumbRef.current
    if (event.button !== 0 || !rail || !thumb || rail.dataset.scrollable !== "true") return

    event.preventDefault()
    // 点击轨道时让滑块中心跳到指针位置；拖动滑块时保留原有抓取点。
    const offset =
      event.target === thumb
        ? event.clientY - thumb.getBoundingClientRect().top
        : thumb.offsetHeight / 2
    dragRef.current = { pointerId: event.pointerId, offset }
    rail.dataset.dragging = "true"
    rail.setPointerCapture(event.pointerId)
    scrollToPointer(event.clientY, offset)
  }

  const handlePointerMove = (event: PointerEvent<HTMLDivElement>) => {
    if (dragRef.current?.pointerId === event.pointerId) {
      scrollToPointer(event.clientY, dragRef.current.offset)
    }
  }

  const handlePointerEnd = (event: PointerEvent<HTMLDivElement>) => {
    const rail = railRef.current
    if (dragRef.current?.pointerId !== event.pointerId || !rail) return
    dragRef.current = null
    delete rail.dataset.dragging
    if (rail.hasPointerCapture(event.pointerId)) rail.releasePointerCapture(event.pointerId)
  }

  return (
    <div className={cn("scroll-region", className)}>
      <Element
        ref={viewportRef}
        className={cn("scroll-region__viewport", viewportClassName)}
        aria-label={ariaLabel}
        tabIndex={0}
        onScroll={syncThumb}
      >
        <div ref={contentRef} className="scroll-region__content">
          {children}
        </div>
      </Element>
      <div
        ref={railRef}
        className="scroll-region__rail"
        aria-hidden="true"
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerEnd}
        onPointerCancel={handlePointerEnd}
        onLostPointerCapture={handlePointerEnd}
      >
        <div ref={thumbRef} className="scroll-region__thumb" />
      </div>
    </div>
  )
}
