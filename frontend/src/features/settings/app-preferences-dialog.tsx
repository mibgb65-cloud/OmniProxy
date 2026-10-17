import { Info, Settings, X } from "lucide-react"
import { Dialog } from "radix-ui"
import { useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { ScrollRegion } from "../../components/common/scroll-region"
import { SidebarLabel } from "../../components/common/sidebar-label"
import { Button } from "../../components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../../components/ui/tabs"
import { Tooltip } from "../../components/ui/tooltip"
import { AboutView } from "../about/about-view"
import { useAppPreferences, type AppPreferencesTab } from "./app-preferences-state"
import { SettingsContent } from "./settings-content"

export function AppPreferencesDialog({ collapsed }: { collapsed: boolean }) {
  const { t } = useTranslation()
  const open = useAppPreferences((state) => state.open)
  const setOpen = useAppPreferences((state) => state.setOpen)
  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Tooltip content={collapsed ? t("preferences.title") : undefined} side="right">
        <Dialog.Trigger asChild>
          <Button
            variant="ghost"
            size="sm"
            className="h-9 w-full justify-start gap-2 px-4 transition-colors has-[>svg]:px-4"
            aria-label={t("preferences.title")}
          >
            <Settings className="size-4 shrink-0" aria-hidden="true" />
            <SidebarLabel collapsed={collapsed}>{t("preferences.title")}</SidebarLabel>
          </Button>
        </Dialog.Trigger>
      </Tooltip>
      {open ? <PreferencesWindow /> : null}
    </Dialog.Root>
  )
}

function PreferencesWindow() {
  const { t } = useTranslation()
  const tab = useAppPreferences((state) => state.tab)
  const setTab = useAppPreferences((state) => state.setTab)
  const activeTab = useRef<HTMLButtonElement>(null)
  const [visited, setVisited] = useState<AppPreferencesTab[]>([tab])
  // 已访问页签保留表单和异步反馈，打开窗口时才挂载，关闭后释放内容。
  if (!visited.includes(tab)) setVisited([...visited, tab])
  return (
    <Dialog.Portal>
      <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm" />
      <Dialog.Content
        className="bg-popover text-popover-foreground fixed top-1/2 left-1/2 z-50 flex max-h-[calc(100dvh-2rem)] w-[calc(100vw-2rem)] max-w-5xl -translate-x-1/2 -translate-y-1/2 flex-col gap-4 rounded-2xl border p-6 shadow-2xl outline-none"
        onOpenAutoFocus={(event) => {
          event.preventDefault()
          activeTab.current?.focus()
        }}
      >
        <header className="flex shrink-0 items-start justify-between gap-4">
          <div className="min-w-0 space-y-1">
            <Dialog.Title className="text-lg font-semibold">{t("preferences.title")}</Dialog.Title>
            <Dialog.Description className="text-muted-foreground text-sm">
              {t("preferences.description")}
            </Dialog.Description>
          </div>
          <Dialog.Close asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              className="shrink-0"
              aria-label={t("preferences.close")}
            >
              <X aria-hidden="true" />
            </Button>
          </Dialog.Close>
        </header>
        <Tabs
          value={tab}
          onValueChange={(value) => {
            if (value === "settings" || value === "about") setTab(value)
          }}
          className="flex min-h-0 flex-1 flex-col gap-4"
        >
          <TabsList className="shrink-0" aria-label={t("preferences.title")}>
            <TabsTrigger value="settings" ref={tab === "settings" ? activeTab : undefined}>
              <Settings aria-hidden="true" />
              {t("nav.settings")}
            </TabsTrigger>
            <TabsTrigger value="about" ref={tab === "about" ? activeTab : undefined}>
              <Info aria-hidden="true" />
              {t("nav.about")}
            </TabsTrigger>
          </TabsList>
          <ScrollRegion
            as="section"
            ariaLabel={t(tab === "settings" ? "nav.settings" : "nav.about")}
            className="min-h-0 flex-1"
            viewportClassName="pr-2"
            pageKey={tab}
          >
            <TabsContent
              value="settings"
              forceMount={visited.includes("settings") ? true : undefined}
              hidden={tab !== "settings"}
            >
              <SettingsContent />
            </TabsContent>
            <TabsContent
              value="about"
              forceMount={visited.includes("about") ? true : undefined}
              hidden={tab !== "about"}
            >
              <AboutView embedded />
            </TabsContent>
          </ScrollRegion>
        </Tabs>
      </Dialog.Content>
    </Dialog.Portal>
  )
}
