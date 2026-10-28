import { useLayoutEffect, useRef, useState, type KeyboardEvent } from "react"
import { ChevronLeft, ChevronRight } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { ProviderConfiguration } from "../../services/omniproxy"
import { ProviderDockCard } from "./provider-dock-card"

export function ProviderSwitcherDock({
  providers,
  accountCounts,
  onSelect,
}: {
  providers: ProviderConfiguration[]
  accountCounts: ReadonlyMap<string, number>
  onSelect: (id: string) => void
}) {
  const { t } = useTranslation()
  const viewportRef = useRef<HTMLDivElement>(null)
  const [edges, setEdges] = useState({ start: true, end: true })
  const providerIds = providers.map((provider) => provider.id).join(",")
  const updateEdges = () => {
    const viewport = viewportRef.current
    if (!viewport) return
    const next = {
      start: viewport.scrollLeft <= 1,
      end: viewport.scrollLeft + viewport.clientWidth >= viewport.scrollWidth - 1,
    }
    setEdges((previous) =>
      previous.start === next.start && previous.end === next.end ? previous : next,
    )
  }

  useLayoutEffect(() => {
    const viewport = viewportRef.current
    if (!viewport) return
    updateEdges()
    const observer = typeof ResizeObserver === "function" ? new ResizeObserver(updateEdges) : null
    observer?.observe(viewport)
    for (const slot of viewport.children) observer?.observe(slot)
    return () => observer?.disconnect()
  }, [providerIds])

  const moveFocus = (event: KeyboardEvent<HTMLDivElement>) => {
    const buttons = Array.from(
      viewportRef.current?.querySelectorAll<HTMLButtonElement>(".bento-dock-card") ?? [],
    )
    const current = buttons.indexOf(event.target as HTMLButtonElement)
    if (current < 0) return
    const next =
      event.key === "ArrowRight"
        ? (current + 1) % buttons.length
        : event.key === "ArrowLeft"
          ? (current - 1 + buttons.length) % buttons.length
          : event.key === "Home"
            ? 0
            : event.key === "End"
              ? buttons.length - 1
              : -1
    if (next < 0) return
    event.preventDefault()
    buttons[next].focus({ preventScroll: true })
    buttons[next].scrollIntoView?.({ block: "nearest", inline: "nearest" })
  }

  const scroll = (direction: number) => {
    const viewport = viewportRef.current
    if (!viewport) return
    const reduced = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches
    viewport.scrollBy({
      left: direction * viewport.clientWidth * 0.75,
      behavior: reduced ? "instant" : "smooth",
    })
  }

  return (
    <div className="bento-dock" role="group" aria-label={t("accounts.dock.label")}>
      <div className="bento-dock-heading">
        <span>{t("accounts.dock.label")}</span>
        <div className="flex gap-1">
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label={t("accounts.dock.previous")}
            disabled={edges.start}
            onClick={() => scroll(-1)}
          >
            <ChevronLeft aria-hidden="true" />
          </Button>
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label={t("accounts.dock.next")}
            disabled={edges.end}
            onClick={() => scroll(1)}
          >
            <ChevronRight aria-hidden="true" />
          </Button>
        </div>
      </div>
      <div
        ref={viewportRef}
        className="bento-dock-viewport"
        onScroll={updateEdges}
        onKeyDown={moveFocus}
      >
        {providers.map((provider) => (
          <div key={provider.id} className="bento-flip bento-dock-slot" data-flip-id={provider.id}>
            <ProviderDockCard
              provider={provider}
              accountCount={accountCounts.get(provider.id) ?? 0}
              onSelect={() => onSelect(provider.id)}
            />
          </div>
        ))}
      </div>
    </div>
  )
}
