import { useCallback, useRef, type MouseEvent as ReactMouseEvent } from "react"
import { flushSync } from "react-dom"
import { applyTheme, themeRevealRadius, type Theme } from "../lib/theme"
import { useUIStore } from "../stores/ui-store"

/*
 * 主题切换的圆形扩散过渡。视觉层在 `components/common/theme-reveal.css`（由标题栏导入），
 * 这里只负责坐标、半径与状态提交。
 */

const revealOriginVars = ["--theme-reveal-x", "--theme-reveal-y", "--theme-reveal-radius"] as const

/** 遵循系统的「减少动画」偏好：此时直接换主题，不做扩散。 */
function prefersReducedMotion(): boolean {
  // jsdom 与部分旧 WebView 没有 matchMedia，缺失即视为未要求减少动画。
  return (
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches
  )
}

/**
 * 键盘激活（Enter／Space）时事件坐标恒为 0；此时用按钮中心作圆心，
 * 否则圆形会从窗口左上角扫过，与用户的操作位置无关。
 */
function revealOrigin(event: ReactMouseEvent<HTMLElement>): { x: number; y: number } {
  if (event.clientX !== 0 || event.clientY !== 0) {
    return { x: event.clientX, y: event.clientY }
  }

  const rect = event.currentTarget.getBoundingClientRect()
  return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 }
}

export type ThemeToggleHandler = (event: ReactMouseEvent<HTMLElement>) => void

/**
 * 在点击位置触发主题切换：支持的运行时用 View Transitions 做圆形扩散，
 * 不支持的运行时退化为普通状态切换。
 */
export function useThemeTransition(): ThemeToggleHandler {
  const toggleTheme = useUIStore((s) => s.toggleTheme)
  // 连续点击会打断上一段过渡并让它 reject；只允许最新一次点击清理自定义属性，
  // 否则旧过渡的清理会抹掉新过渡的圆心与半径。
  const revealId = useRef(0)

  return useCallback<ThemeToggleHandler>(
    (event) => {
      const nextTheme: Theme = useUIStore.getState().theme === "dark" ? "light" : "dark"

      // 更新回调里先同步改 DOM 再提交状态：新快照在回调返回时拍摄，早于 React 的下一次渲染；
      // flushSync 让标题栏图标与新快照在同一帧内一致。
      const commit = () => {
        applyTheme(nextTheme)
        flushSync(toggleTheme)
      }

      if (typeof document.startViewTransition !== "function" || prefersReducedMotion()) {
        commit()
        return
      }

      const id = (revealId.current += 1)
      const { x, y } = revealOrigin(event)
      const root = document.documentElement
      const radius = themeRevealRadius(x, y, window.innerWidth, window.innerHeight)
      root.style.setProperty("--theme-reveal-x", `${x}px`)
      root.style.setProperty("--theme-reveal-y", `${y}px`)
      root.style.setProperty("--theme-reveal-radius", `${radius}px`)

      const clearRevealOrigin = () => {
        if (revealId.current !== id) return
        for (const name of revealOriginVars) root.style.removeProperty(name)
      }

      try {
        const transition = document.startViewTransition(commit)
        // 过渡被跳过或被后续切换打断时 finished 会 reject，两种情况都要清理自定义属性。
        void transition.finished.then(clearRevealOrigin, clearRevealOrigin)
      } catch {
        // 浏览器拒绝启动过渡（例如正在进行的过渡无法打断）时仍要完成主题切换。
        commit()
        clearRevealOrigin()
      }
    },
    [toggleTheme],
  )
}
