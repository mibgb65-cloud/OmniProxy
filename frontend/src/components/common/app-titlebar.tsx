import { useEffect, useState } from "react"
import { Events, Window } from "@wailsio/runtime"
import { Copy, Minus, Moon, Square, Sun, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { useUIStore } from "../../stores/ui-store"

const controlClass =
  "hover:bg-sidebar-accent focus-visible:ring-ring flex h-11 w-11 shrink-0 cursor-pointer items-center justify-center outline-none focus-visible:ring-2 focus-visible:ring-inset"

export function AppTitlebar() {
  const { t } = useTranslation()
  const theme = useUIStore((s) => s.theme)
  const toggleTheme = useUIStore((s) => s.toggleTheme)
  const [isMaximised, setIsMaximised] = useState(false)
  const [actionFailed, setActionFailed] = useState(false)

  useEffect(() => {
    let mounted = true
    void Window.IsMaximised()
      .then((value) => {
        if (mounted) setIsMaximised(value)
      })
      .catch(() => {
        if (mounted) setActionFailed(true)
      })

    const offMaximise = Events.On(Events.Types.Common.WindowMaximise, () => setIsMaximised(true))
    const offRestore = Events.On(Events.Types.Common.WindowUnMaximise, () => setIsMaximised(false))
    return () => {
      mounted = false
      offMaximise()
      offRestore()
    }
  }, [])

  const runWindowAction = (action: () => Promise<unknown>) => {
    setActionFailed(false)
    void action().catch(() => setActionFailed(true))
  }

  const toggleMaximise = () => {
    runWindowAction(async () => {
      await Window.ToggleMaximise()
      setIsMaximised(await Window.IsMaximised())
    })
  }

  const themeLabel = theme === "dark" ? t("window.switchToLight") : t("window.switchToDark")

  return (
    <header className="window-titlebar border-border bg-sidebar text-sidebar-foreground flex h-11 w-full shrink-0 items-center border-b pl-3">
      <div className="flex h-full min-w-0 flex-1 items-center gap-2" onDoubleClick={toggleMaximise}>
        <img src="/favicon.svg" alt="" className="size-5 shrink-0 rounded" />
        <span className="truncate text-xs font-semibold tracking-wide">{t("window.title")}</span>
      </div>

      {actionFailed ? (
        <span role="alert" className="text-destructive mr-2 text-xs">
          {t("window.actionFailed")}
        </span>
      ) : null}

      <div className="flex h-full shrink-0 items-center">
        <button
          type="button"
          className={controlClass}
          aria-label={themeLabel}
          title={themeLabel}
          onClick={toggleTheme}
        >
          {theme === "dark" ? <Sun className="size-4" /> : <Moon className="size-4" />}
        </button>
        <span className="bg-border mx-1 h-4 w-px" aria-hidden="true" />
        <button
          type="button"
          className={controlClass}
          aria-label={t("window.minimise")}
          title={t("window.minimise")}
          onClick={() => runWindowAction(() => Window.Minimise())}
        >
          <Minus className="size-4" />
        </button>
        <button
          type="button"
          className={controlClass}
          aria-label={isMaximised ? t("window.restore") : t("window.maximise")}
          title={isMaximised ? t("window.restore") : t("window.maximise")}
          onClick={toggleMaximise}
        >
          {isMaximised ? <Copy className="size-4" /> : <Square className="size-3.5" />}
        </button>
        <button
          type="button"
          className={`${controlClass} hover:bg-destructive hover:text-white`}
          aria-label={t("window.close")}
          title={t("window.close")}
          onClick={() => runWindowAction(() => Window.Close())}
        >
          <X className="size-4" />
        </button>
      </div>
    </header>
  )
}
