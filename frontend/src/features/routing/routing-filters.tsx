import { SelectField } from "../../components/ui/select-field"
import { useTranslation } from "react-i18next"
import { Input } from "../../components/ui/input"

export function RoutingFilters({
  search,
  filter,
  count,
  onSearch,
  onFilter,
}: {
  search: string
  filter: string
  count: number
  onSearch: (value: string) => void
  onFilter: (value: string) => void
}) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-wrap items-center gap-3">
      <Input
        aria-label={t("routing.search")}
        placeholder={t("routing.searchPlaceholder")}
        className="max-w-sm"
        maxLength={128}
        value={search}
        onChange={(event) => onSearch(event.target.value)}
      />
      <SelectField
        aria-label={t("routing.filter")}
        className="w-auto min-w-36"
        value={filter}
        onValueChange={onFilter}
        options={[
          { value: "all", label: t("routing.all") },
          { value: "enabled", label: t("routing.enabled") },
          { value: "disabled", label: t("routing.disabled") },
          { value: "invalid", label: t("routing.invalid") },
        ]}
      />
      <p className="text-muted-foreground text-xs">{t("routing.count", { count })}</p>
    </div>
  )
}
