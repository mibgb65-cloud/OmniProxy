import { useTranslation } from "react-i18next"
import { Input } from "../../components/ui/input"
import { SelectField } from "../../components/ui/select-field"
import { quotaStatuses } from "./quota-model"
import type { QuotaOverview } from "./use-quota-overview"

export function QuotaFilters({ data }: { data: QuotaOverview }) {
  const { t } = useTranslation()
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-[2fr_1fr_1fr]">
      <label className="text-muted-foreground space-y-1.5 text-xs md:col-span-2 xl:col-span-1">
        <span>{t("quota.search")}</span>
        <Input
          type="search"
          maxLength={200}
          value={data.search}
          placeholder={t("quota.searchPlaceholder")}
          onChange={(event) => data.setSearch(event.target.value)}
          disabled={!data.ready}
        />
      </label>
      <div className="space-y-1.5">
        <p className="text-muted-foreground text-xs">{t("quota.provider")}</p>
        <SelectField
          aria-label={t("quota.provider")}
          value={data.provider}
          onValueChange={data.setProvider}
          disabled={!data.ready}
          options={[
            { value: "", label: t("quota.allProviders") },
            ...data.providers.map((item) => ({ value: item.id, label: item.name })),
          ]}
        />
      </div>
      <div className="space-y-1.5">
        <p className="text-muted-foreground text-xs">{t("quota.status")}</p>
        <SelectField
          aria-label={t("quota.status")}
          value={data.status}
          onValueChange={data.setStatus}
          disabled={!data.ready}
          options={[
            { value: "", label: t("quota.allStatuses") },
            ...quotaStatuses.map((status) => ({
              value: status,
              label: t(`quota.status.${status}`),
            })),
          ]}
        />
      </div>
    </div>
  )
}
