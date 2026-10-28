import { render, waitFor } from "@testing-library/react"
import { useState } from "react"
import { afterEach, beforeEach, describe, expect, it } from "vitest"
import { stubNavRowMetrics } from "../../test/nav-row-metrics"
import { SidebarNavIndicator } from "./sidebar-nav-indicator"

function Harness({ active }: { active: number | null }) {
  const [list, setList] = useState<HTMLUListElement | null>(null)
  return (
    <div className="app-sidebar-nav">
      <SidebarNavIndicator list={list} />
      <ul ref={setList}>
        {["仪表盘", "账号", "提供商"].map((label, index) => (
          <li key={label}>
            <a href={`#${label}`} data-active={index === active ? "true" : undefined}>
              {label}
            </a>
          </li>
        ))}
      </ul>
    </div>
  )
}

function indicator(container: HTMLElement) {
  return container.querySelector<HTMLElement>(".sidebar-nav-indicator")
}

describe("SidebarNavIndicator", () => {
  let restoreMetrics: () => void

  beforeEach(() => {
    restoreMetrics = stubNavRowMetrics()
  })

  afterEach(() => {
    restoreMetrics()
  })

  it("positions the background on the active row and moves it instead of recreating it", async () => {
    const { container, rerender } = render(<Harness active={1} />)

    expect(indicator(container)?.style.transform).toBe("translateY(40px)")
    expect(indicator(container)?.style.height).toBe("36px")
    const background = indicator(container)

    rerender(<Harness active={2} />)
    await waitFor(() => expect(background?.style.transform).toBe("translateY(80px)"))
    // 同一个背景元素在移动，旧块不会先消失再让新块出现。
    expect(indicator(container)).toBe(background)
  })

  it("renders nothing when the route matches no navigation item", async () => {
    const { container, rerender } = render(<Harness active={0} />)
    expect(indicator(container)).not.toBeNull()

    rerender(<Harness active={null} />)
    await waitFor(() => expect(indicator(container)).toBeNull())
  })

  it("keeps the slide transition disabled until the first measurement settles", async () => {
    const { container } = render(<Harness active={0} />)

    // 首帧先落位、但不带过渡，避免从容器顶部滑下来的误触发动画。
    expect(indicator(container)).toHaveAttribute("data-measured", "false")
    expect(indicator(container)?.style.transform).toBe("translateY(0px)")
    await waitFor(() => expect(indicator(container)).toHaveAttribute("data-measured", "true"))
  })
})
