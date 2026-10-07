import { LayoutGrid, List } from "lucide-react"
import { ToggleGroup } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { isAccountView, useUIStore } from "../../stores/ui-store"

export function AccountViewToggle() {
  const { t } = useTranslation()
  const view = useUIStore((state) => state.accountView)
  const setView = useUIStore((state) => state.setAccountView)
  return (
    <ToggleGroup.Root
      type="single"
      value={view}
      aria-label={t("accounts.view.label")}
      onValueChange={(value) => {
        if (isAccountView(value)) setView(value)
      }}
      className="bg-muted flex shrink-0 items-center gap-1 rounded-lg p-1"
    >
      <ToggleGroup.Item value="cards" aria-label={t("accounts.view.cardsLabel")} asChild>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          title={t("accounts.view.cardsLabel")}
          className="data-[state=on]:bg-background cursor-pointer data-[state=on]:shadow-sm"
        >
          <LayoutGrid className="size-4" aria-hidden="true" />
          {t("accounts.view.cards")}
        </Button>
      </ToggleGroup.Item>
      <ToggleGroup.Item value="list" aria-label={t("accounts.view.listLabel")} asChild>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          title={t("accounts.view.listLabel")}
          className="data-[state=on]:bg-background cursor-pointer data-[state=on]:shadow-sm"
        >
          <List className="size-4" aria-hidden="true" />
          {t("accounts.view.list")}
        </Button>
      </ToggleGroup.Item>
    </ToggleGroup.Root>
  )
}
