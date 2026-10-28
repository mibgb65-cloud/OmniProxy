import { fireEvent, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { defaultTheme, themeRevealRadius, type Theme } from "../lib/theme"
import { useUIStore } from "../stores/ui-store"
import { useThemeTransition } from "./use-theme-transition"

function ThemeToggleHarness() {
  const toggleThemeWithReveal = useThemeTransition()
  return (
    <button type="button" onClick={toggleThemeWithReveal}>
      切换主题
    </button>
  )
}

/** jsdom 没有 View Transitions，测试用一个可控替身观察调用参数与回调时机。 */
function setStartViewTransition(value: unknown) {
  Object.defineProperty(document, "startViewTransition", {
    configurable: true,
    writable: true,
    value,
  })
}

function installViewTransition() {
  // 每次调用一个独立的 finished，便于模拟「上一段过渡被下一次点击打断」。
  const settles: Array<(error?: unknown) => void> = []
  const startViewTransition = vi.fn((callback: () => void) => {
    callback()
    const finished = new Promise<void>((resolve, reject) => {
      settles.push((error) => (error ? reject(error) : resolve()))
    })
    return { finished, ready: Promise.resolve(), updateCallbackDone: Promise.resolve() }
  })
  setStartViewTransition(startViewTransition)

  return { startViewTransition, settles }
}

/** jsdom 没有 matchMedia，注入一个只对「减少动画」查询命中的替身。 */
function stubReducedMotion() {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: (query: string) =>
      ({ matches: query.includes("prefers-reduced-motion"), media: query }) as MediaQueryList,
  })
}

function setViewport(width: number, height: number) {
  for (const [name, value] of [
    ["innerWidth", width],
    ["innerHeight", height],
  ] as const) {
    Object.defineProperty(window, name, { configurable: true, writable: true, value })
  }
}

function revealVars() {
  const style = document.documentElement.style
  return {
    x: style.getPropertyValue("--theme-reveal-x"),
    y: style.getPropertyValue("--theme-reveal-y"),
    radius: style.getPropertyValue("--theme-reveal-radius"),
  }
}

function setTheme(theme: Theme) {
  useUIStore.setState({ theme })
  document.documentElement.classList.toggle("dark", theme === "dark")
}

describe("useThemeTransition", () => {
  beforeEach(() => {
    localStorage.clear()
    setTheme(defaultTheme)
    setViewport(1200, 800)
  })

  afterEach(() => {
    Reflect.deleteProperty(document, "startViewTransition")
    Reflect.deleteProperty(window, "matchMedia")
    vi.restoreAllMocks()
  })

  it("switches the theme without an animation when View Transitions is unavailable", () => {
    render(<ThemeToggleHarness />)

    fireEvent.click(screen.getByRole("button", { name: "切换主题" }), { clientX: 100, clientY: 50 })

    expect(useUIStore.getState().theme).toBe("light")
    expect(document.documentElement).not.toHaveClass("dark")
    expect(revealVars()).toEqual({ x: "", y: "", radius: "" })
  })

  it("reveals the new theme from the click position with the farthest-corner radius", async () => {
    const { startViewTransition, settles } = installViewTransition()
    render(<ThemeToggleHarness />)

    fireEvent.click(screen.getByRole("button", { name: "切换主题" }), { clientX: 100, clientY: 50 })

    expect(startViewTransition).toHaveBeenCalledOnce()
    // 主题在过渡的更新回调里提交，新快照因此带的是新主题。
    expect(useUIStore.getState().theme).toBe("light")
    expect(document.documentElement).not.toHaveClass("dark")
    expect(revealVars()).toEqual({
      x: "100px",
      y: "50px",
      radius: `${themeRevealRadius(100, 50, 1200, 800)}px`,
    })

    settles[0]()
    await Promise.resolve()
    expect(revealVars()).toEqual({ x: "", y: "", radius: "" })
  })

  it("reveals from the button centre when activated from the keyboard", () => {
    installViewTransition()
    render(<ThemeToggleHarness />)
    const button = screen.getByRole("button", { name: "切换主题" })
    vi.spyOn(button, "getBoundingClientRect").mockReturnValue({
      left: 100,
      top: 10,
      width: 44,
      height: 44,
      right: 144,
      bottom: 54,
      x: 100,
      y: 10,
      toJSON: () => ({}),
    })

    fireEvent.click(button, { clientX: 0, clientY: 0 })

    expect(revealVars()).toMatchObject({ x: "122px", y: "32px" })
  })

  it("cleans the reveal origin when the transition is skipped", async () => {
    const { settles } = installViewTransition()
    render(<ThemeToggleHarness />)

    fireEvent.click(screen.getByRole("button", { name: "切换主题" }), { clientX: 10, clientY: 10 })
    settles[0](new Error("skipped"))
    await Promise.resolve()
    await Promise.resolve()

    expect(revealVars()).toEqual({ x: "", y: "", radius: "" })
    expect(useUIStore.getState().theme).toBe("light")
  })

  it("keeps the newest reveal origin when a click interrupts the previous transition", async () => {
    const { settles } = installViewTransition()
    render(<ThemeToggleHarness />)
    const button = screen.getByRole("button", { name: "切换主题" })

    fireEvent.click(button, { clientX: 10, clientY: 10 })
    fireEvent.click(button, { clientX: 300, clientY: 200 })
    expect(revealVars()).toMatchObject({ x: "300px", y: "200px" })

    // 上一段过渡被打断后 reject，不能抹掉最新一次点击的圆心与半径。
    settles[0](new Error("interrupted"))
    await Promise.resolve()
    await Promise.resolve()
    expect(revealVars()).toMatchObject({ x: "300px", y: "200px" })

    settles[1]()
    await Promise.resolve()
    expect(revealVars()).toEqual({ x: "", y: "", radius: "" })
  })

  it("switches instantly when the system asks for reduced motion", () => {
    const { startViewTransition } = installViewTransition()
    stubReducedMotion()
    render(<ThemeToggleHarness />)

    fireEvent.click(screen.getByRole("button", { name: "切换主题" }), { clientX: 100, clientY: 50 })

    expect(startViewTransition).not.toHaveBeenCalled()
    expect(useUIStore.getState().theme).toBe("light")
  })

  it("still switches the theme when the browser refuses to start a transition", () => {
    setStartViewTransition(
      vi.fn(() => {
        throw new Error("transition not allowed")
      }),
    )
    render(<ThemeToggleHarness />)

    fireEvent.click(screen.getByRole("button", { name: "切换主题" }), { clientX: 100, clientY: 50 })

    expect(useUIStore.getState().theme).toBe("light")
    expect(document.documentElement).not.toHaveClass("dark")
    expect(revealVars()).toEqual({ x: "", y: "", radius: "" })
  })
})
