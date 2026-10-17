import { useTranslation } from "react-i18next"
import { formatBytes } from "../../lib/format"
import type { StorageInfo } from "../../services/omniproxy"

const CATEGORIES = [
  { name: "database", label: "settings.storage.database", color: "bg-blue-500" },
  { name: "logs", label: "settings.storage.logs", color: "bg-emerald-500" },
  { name: "cache", label: "settings.storage.cache", color: "bg-amber-500" },
  { name: "other", label: "settings.storage.other", color: "bg-violet-500" },
] as const

export function StorageUsageChart({ data }: { data: StorageInfo }) {
  const { t } = useTranslation()
  const total = Math.max(0, data.totalBytes)
  const [amount, unit] = formatBytes(total).split(" ")
  const categories = CATEGORIES.map((category) => {
    const bytes = Math.max(0, data.items?.find((item) => item.name === category.name)?.bytes ?? 0)
    return { ...category, bytes, fraction: total > 0 ? bytes / total : 0 }
  })

  return (
    <figure className="@container">
      <figcaption className="sr-only">{t("settings.storage.breakdown")}</figcaption>
      <div className="grid gap-4 border-b pb-4 @xl:grid-cols-[minmax(180px,240px)_minmax(0,1fr)] @xl:gap-6">
        <div className="bg-muted/25 flex flex-col justify-center rounded-lg border px-4 py-4">
          <span className="text-muted-foreground text-xs">{t("settings.storage.total")}</span>
          <div className="mt-1 flex items-baseline gap-1.5 tabular-nums">
            <strong className="text-3xl font-semibold tracking-tight">{amount}</strong>
            <span className="text-muted-foreground text-sm">{unit}</span>
          </div>
        </div>

        <div className="flex min-w-0 flex-col justify-center gap-4">
          <div
            className="bg-muted flex h-2.5 w-full overflow-hidden rounded-full"
            role="img"
            aria-label={t("settings.storage.chartLabel")}
          >
            {categories
              .filter((category) => category.bytes > 0)
              .map((category) => (
                <span
                  key={category.name}
                  className={`h-full shrink-0 ${category.color}`}
                  style={{ width: `${category.fraction * 100}%` }}
                  aria-hidden="true"
                />
              ))}
          </div>
          <dl className="grid grid-cols-2 gap-x-4 gap-y-3 @lg:grid-cols-4">
            {categories.map((category) => (
              <div key={category.name} className="min-w-0">
                <dt className="text-muted-foreground flex items-center gap-1.5 text-xs">
                  <span
                    className={`size-2 shrink-0 rounded-full ${category.color}`}
                    aria-hidden="true"
                  />
                  {t(category.label)}
                </dt>
                <dd className="mt-1 text-sm font-medium tabular-nums">
                  {formatBytes(category.bytes)}
                  <span className="text-muted-foreground ml-1 text-xs font-normal">
                    {formatPercentage(category.fraction)}
                  </span>
                </dd>
              </div>
            ))}
          </dl>
        </div>
      </div>
    </figure>
  )
}

function formatPercentage(fraction: number): string {
  if (fraction > 0 && fraction < 0.01) {
    return "<1%"
  }
  return `${Math.round(fraction * 100)}%`
}
