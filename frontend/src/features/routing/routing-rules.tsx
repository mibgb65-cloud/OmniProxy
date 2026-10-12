import { useState } from "react"
import { useTranslation } from "react-i18next"
import type { RouteItem } from "../../services/routing"
import { RoutingFilters } from "./routing-filters"
import { RoutingTable } from "./routing-table"

export function RoutingRules({
  items,
  busy,
  onEdit,
  onDelete,
  onToggle,
}: {
  items: RouteItem[]
  busy: boolean
  onEdit: (item: RouteItem) => void
  onDelete: (item: RouteItem) => void
  onToggle: (id: string, enabled: boolean) => void
}) {
  const { t } = useTranslation()
  const [search, setSearch] = useState("")
  const [filter, setFilter] = useState("all")
  const visible = items.filter((item) => {
    const matches = [item.rule.alias, item.rule.model, item.providerName].some((value) =>
      value.toLowerCase().includes(search.toLowerCase()),
    )
    return (
      matches &&
      (filter === "all" ||
        (filter === "enabled" && item.rule.enabled) ||
        (filter === "disabled" && !item.rule.enabled) ||
        (filter === "invalid" && !!item.errorCode))
    )
  })
  return (
    <div className="min-w-0 space-y-5">
      <RoutingFilters
        search={search}
        filter={filter}
        count={items.length}
        onSearch={setSearch}
        onFilter={setFilter}
      />
      {items.length === 0 ? (
        <div className="bg-muted/10 space-y-2 rounded-xl border border-dashed p-5">
          <h2 className="text-base font-semibold">{t("routing.empty")}</h2>
          <p className="text-muted-foreground max-w-md text-sm">{t("routing.emptyHint")}</p>
        </div>
      ) : visible.length ? (
        <RoutingTable
          items={visible}
          busy={busy}
          onEdit={onEdit}
          onDelete={onDelete}
          onToggle={onToggle}
        />
      ) : (
        <p className="text-muted-foreground rounded-xl border p-6 text-center text-sm">
          {t("routing.noMatches")}
        </p>
      )}
      <p className="text-muted-foreground text-xs leading-5">{t("routing.policy")}</p>
    </div>
  )
}
