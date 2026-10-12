import { Plus, RefreshCw } from "lucide-react"
import type { RefObject } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"

export function RoutingHeader({
  fetching,
  disabled,
  addRef,
  onRefresh,
  onAdd,
}: {
  fetching: boolean
  disabled: boolean
  addRef: RefObject<HTMLButtonElement | null>
  onRefresh: () => void
  onAdd: () => void
}) {
  const { t } = useTranslation()
  return (
    <header className="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">{t("pages.routing.title")}</h1>
      </div>
      <div className="flex gap-2">
        <Button
          variant="outline"
          size="icon"
          aria-label={t("routing.refresh")}
          disabled={fetching}
          onClick={onRefresh}
        >
          <RefreshCw aria-hidden="true" className={fetching ? "motion-safe:animate-spin" : ""} />
        </Button>
        <Button ref={addRef} disabled={disabled} onClick={onAdd}>
          <Plus aria-hidden="true" />
          {t("routing.add")}
        </Button>
      </div>
    </header>
  )
}
