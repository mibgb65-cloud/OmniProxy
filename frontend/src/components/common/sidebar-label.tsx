import type { CSSProperties } from "react"
import { cn } from "../../lib/utils"
import "./sidebar-label.css"

const segmenter = new Intl.Segmenter(undefined, { granularity: "grapheme" })
const CHARACTER_STAGGER_MS = 32
const MAX_STAGGER_MS = 150
const ENTER_DELAY_MS = 35

type CharacterStyle = CSSProperties & {
  "--enter-delay": string
  "--exit-delay": string
}

interface SidebarLabelProps {
  children: string
  collapsed: boolean
  className?: string
}

export function SidebarLabel({ children, collapsed, className }: SidebarLabelProps) {
  // 按字素拆分，避免组合字符被拆开；限制总延迟，让长英文和短中文都跟得上侧栏。
  const characters = Array.from(segmenter.segment(children), ({ segment }) => segment)
  const stagger = Math.min(
    CHARACTER_STAGGER_MS,
    MAX_STAGGER_MS / Math.max(1, characters.length - 1),
  )

  return (
    <span
      className={cn("sidebar-label inline-flex shrink-0 whitespace-pre", className)}
      data-collapsed={collapsed}
      aria-hidden={collapsed}
    >
      <span className="sr-only">{children}</span>
      {characters.map((character, index) => {
        const style: CharacterStyle = {
          "--enter-delay": `${ENTER_DELAY_MS + index * stagger}ms`,
          "--exit-delay": `${(characters.length - 1 - index) * stagger}ms`,
        }
        return (
          <span
            key={`${index}-${character}`}
            className="sidebar-label-character"
            aria-hidden="true"
            style={style}
          >
            {character}
          </span>
        )
      })}
    </span>
  )
}
