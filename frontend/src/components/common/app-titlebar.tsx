import { Tooltip } from "../ui/tooltip"
import { useEffect, useState } from "react"
import { Events, Window } from "@wailsio/runtime"
import { Copy, Minus, Moon, Square, Sun, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { useUIStore } from "../../stores/ui-store"
import { LanguageSwitcher } from "./language-switcher"

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
      {/* 品牌标识只出现在左上角侧栏品牌栏；这里留白仅作拖动区，双击最大化。 */}
      <div
        className="flex h-full min-w-0 flex-1 items-center"
        data-testid="titlebar-drag-region"
        onDoubleClick={toggleMaximise}
      />

      {actionFailed ? (
        <span role="alert" className="text-destructive mr-2 text-xs">
          {t("window.actionFailed")}
        </span>
      ) : null}

      <div className="flex h-full shrink-0 items-center">
        <LanguageSwitcher />
        <Tooltip content={themeLabel}>
          <button
            type="button"
            className={controlClass}
            aria-label={themeLabel}
            onClick={toggleTheme}
          >
            {theme === "dark" ? <Sun className="size-4" /> : <Moon className="size-4" />}
          </button>
        </Tooltip>
        <span className="bg-border mx-1 h-4 w-px" aria-hidden="true" />
        <Tooltip content={t("window.minimise")}>
          <button
            type="button"
            className={controlClass}
            aria-label={t("window.minimise")}
            onClick={() => runWindowAction(() => Window.Minimise())}
          >
            <Minus className="size-4" />
          </button>
        </Tooltip>
        <Tooltip content={isMaximised ? t("window.restore") : t("window.maximise")}>
          <button
            type="button"
            className={controlClass}
            aria-label={isMaximised ? t("window.restore") : t("window.maximise")}
            onClick={toggleMaximise}
          >
            {isMaximised ? <Copy className="size-4" /> : <Square className="size-3.5" />}
          </button>
        </Tooltip>
        <Tooltip content={t("window.close")}>
          <button
            type="button"
            className={`${controlClass} hover:bg-destructive hover:text-white`}
            aria-label={t("window.close")}
            onClick={() => runWindowAction(() => Window.Close())}
          >
            <X className="size-4" />
          </button>
        </Tooltip>
      </div>
    </header>
  )
}
