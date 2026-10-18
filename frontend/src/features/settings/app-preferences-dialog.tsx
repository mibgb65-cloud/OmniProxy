import { Database, Info, RefreshCw, ScrollText, Settings, X } from "lucide-react"
import { Dialog } from "radix-ui"
import { useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { ScrollRegion } from "../../components/common/scroll-region"
import { SidebarLabel } from "../../components/common/sidebar-label"
import { Button } from "../../components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../../components/ui/tabs"
import { Tooltip } from "../../components/ui/tooltip"
import { AboutView } from "../about/about-view"
import { UpdateCard } from "../about/update-card"
import { UpdateBadge } from "../app-update/update-badge"
import { useAppPreferences, type AppPreferencesTab } from "./app-preferences-state"
import { SettingsContent } from "./settings-content"

const SECTIONS = [
  { value: "settings", label: "preferences.general", icon: Settings },
  { value: "activity", label: "preferences.activity", icon: ScrollText },
  { value: "storage", label: "preferences.storage", icon: Database },
  { value: "updates", label: "about.update.title", icon: RefreshCw },
  { value: "about", label: "nav.about", icon: Info },
] as const

export function AppPreferencesDialog({ collapsed }: { collapsed: boolean }) {
  const { t } = useTranslation()
  const open = useAppPreferences((state) => state.open)
  const setOpen = useAppPreferences((state) => state.setOpen)
  return (
    <div className="relative flex h-9 w-full items-center gap-1">
      <Dialog.Root open={open} onOpenChange={setOpen}>
        <Tooltip content={collapsed ? t("preferences.title") : undefined} side="right">
          <Dialog.Trigger asChild>
            <Button
              variant="ghost"
              size="sm"
              className="h-9 min-w-0 flex-1 justify-start gap-2 px-4 transition-colors has-[>svg]:px-4"
              aria-label={t("preferences.title")}
            >
              <Settings className="size-4 shrink-0" aria-hidden="true" />
              <SidebarLabel collapsed={collapsed}>{t("preferences.title")}</SidebarLabel>
            </Button>
          </Dialog.Trigger>
        </Tooltip>
        {open ? <PreferencesWindow /> : null}
      </Dialog.Root>
      <UpdateBadge collapsed={collapsed} onOpen={() => setOpen(false)} />
    </div>
  )
}

function PreferencesWindow() {
  const { t } = useTranslation()
  const tab = useAppPreferences((state) => state.tab)
  const setTab = useAppPreferences((state) => state.setTab)
  const activeTab = useRef<HTMLButtonElement>(null)
  const [visited, setVisited] = useState<AppPreferencesTab[]>([tab])
  const currentSection = SECTIONS.find((section) => section.value === tab)!
  // 已访问页签保留表单和异步反馈，打开窗口时才挂载，关闭后释放内容。
  if (!visited.includes(tab)) setVisited([...visited, tab])
  return (
    <Dialog.Portal>
      <Dialog.Overlay className="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm" />
      <Dialog.Content
        className="bg-background text-foreground fixed top-1/2 left-1/2 z-50 h-[min(36rem,calc(100dvh-3rem))] w-[calc(100vw-3rem)] max-w-4xl -translate-x-1/2 -translate-y-1/2 overflow-hidden rounded-2xl border shadow-2xl outline-none"
        onOpenAutoFocus={(event) => {
          event.preventDefault()
          activeTab.current?.focus()
        }}
      >
        <Dialog.Description className="sr-only">{t("preferences.description")}</Dialog.Description>
        <Tabs
          value={tab}
          orientation="vertical"
          onValueChange={(value) => {
            const section = SECTIONS.find((section) => section.value === value)
            if (section) setTab(section.value)
          }}
          className="grid h-full min-h-0 grid-cols-[9rem_minmax(0,1fr)] sm:grid-cols-[11.5rem_minmax(0,1fr)]"
        >
          <aside className="flex min-h-0 flex-col border-r px-3 py-5">
            <Dialog.Title className="text-muted-foreground mb-5 px-3 text-sm font-medium">
              {t("preferences.title")}
            </Dialog.Title>
            <TabsList
              className="min-h-0 flex-col flex-nowrap gap-1 overflow-y-auto border-0"
              aria-label={t("preferences.title")}
            >
              {SECTIONS.map(({ value, label, icon: Icon }) => (
                <TabsTrigger
                  key={value}
                  value={value}
                  ref={tab === value ? activeTab : undefined}
                  className="data-[state=active]:bg-muted data-[state=active]:text-foreground min-h-10 w-full shrink-0 justify-start rounded-lg border-0 px-3 py-2.5 text-left"
                >
                  <Icon aria-hidden="true" />
                  {t(label)}
                </TabsTrigger>
              ))}
            </TabsList>
          </aside>
          <div className="bg-muted/25 flex min-h-0 min-w-0 flex-col">
            <header className="flex shrink-0 items-center justify-between gap-3 px-5 pt-4 pb-3 sm:px-6">
              <h2 className="text-sm font-semibold">{t(currentSection.label)}</h2>
              <Dialog.Close asChild>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="text-muted-foreground shrink-0"
                  aria-label={t("preferences.close")}
                >
                  <X aria-hidden="true" />
                </Button>
              </Dialog.Close>
            </header>
            {/* 底部留白随内容滚动，避免视口在窗口边框上方提前截断卡片。 */}
            <ScrollRegion
              as="section"
              ariaLabel={t(currentSection.label)}
              className="min-h-0 flex-1 pl-5 sm:pl-6"
              viewportClassName="pr-2 sm:pr-3"
              pageKey={`preferences-${tab}`}
            >
              {SECTIONS.map(({ value }) => (
                <TabsContent
                  key={value}
                  value={value}
                  className="pb-5"
                  forceMount={visited.includes(value) ? true : undefined}
                  hidden={tab !== value}
                >
                  {value === "about" ? (
                    <AboutView embedded />
                  ) : value === "updates" ? (
                    <UpdateCard />
                  ) : (
                    <SettingsContent section={value} />
                  )}
                </TabsContent>
              ))}
            </ScrollRegion>
          </div>
        </Tabs>
      </Dialog.Content>
    </Dialog.Portal>
  )
}
