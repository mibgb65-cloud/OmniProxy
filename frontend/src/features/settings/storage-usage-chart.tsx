import { useTranslation } from "react-i18next"
import { formatBytes } from "../../lib/format"
import type { StorageInfo } from "../../services/omniproxy"

const RADIUS = 62
const CIRCUMFERENCE = 2 * Math.PI * RADIUS
const CATEGORIES = [
  { name: "database", label: "settings.storage.database", color: "var(--chart-1)" },
  { name: "logs", label: "settings.storage.logs", color: "var(--chart-2)" },
  { name: "cache", label: "settings.storage.cache", color: "var(--chart-3)" },
  { name: "other", label: "settings.storage.other", color: "var(--chart-4)" },
] as const

export function StorageUsageChart({ data }: { data: StorageInfo }) {
  const { t } = useTranslation()
  const total = Math.max(0, data.totalBytes)

  const categories = CATEGORIES.reduce<
    ((typeof CATEGORIES)[number] & { bytes: number; fraction: number; offset: number })[]
  >((segments, category) => {
    const bytes = Math.max(0, data.items?.find((item) => item.name === category.name)?.bytes ?? 0)
    const fraction = total > 0 ? bytes / total : 0
    const offset = segments.reduce((sum, segment) => sum + segment.fraction, 0)
    return [...segments, { ...category, bytes, fraction, offset }]
  }, [])

  return (
    <figure className="bg-muted/25 rounded-lg border p-4">
      <figcaption className="font-medium">{t("settings.storage.breakdown")}</figcaption>
      <div className="mt-4 flex flex-col items-center gap-6 sm:flex-row">
        <div
          className="relative size-44 shrink-0"
          role="img"
          aria-label={t("settings.storage.chartLabel")}
        >
          <svg viewBox="0 0 160 160" className="size-full" aria-hidden="true">
            <circle cx="80" cy="80" r={RADIUS} fill="none" stroke="var(--muted)" strokeWidth="16" />
            {categories
              .filter((category) => category.bytes > 0)
              .map((category) => (
                <circle
                  key={category.name}
                  cx="80"
                  cy="80"
                  r={RADIUS}
                  fill="none"
                  stroke={category.color}
                  strokeWidth="16"
                  strokeDasharray={`${category.fraction * CIRCUMFERENCE} ${CIRCUMFERENCE}`}
                  strokeDashoffset={-category.offset * CIRCUMFERENCE}
                  transform="rotate(-90 80 80)"
                />
              ))}
          </svg>
          <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
            <span className="text-muted-foreground text-xs">{t("settings.storage.total")}</span>
            <strong className="mt-1 text-xl tabular-nums">{formatBytes(total)}</strong>
          </div>
        </div>
        <dl className="grid w-full min-w-0 grid-cols-2 gap-x-5 gap-y-4">
          {categories.map((category) => (
            <div key={category.name} className="min-w-0">
              <dt className="text-muted-foreground flex items-center gap-2 text-xs">
                <span
                  className="size-2.5 shrink-0 rounded-sm"
                  style={{ backgroundColor: category.color }}
                  aria-hidden="true"
                />
                {t(category.label)}
              </dt>
              <dd className="mt-1 text-sm font-medium tabular-nums">
                {formatBytes(category.bytes)}
                <span className="text-muted-foreground ml-2 text-xs font-normal">
                  {formatPercentage(category.fraction)}
                </span>
              </dd>
            </div>
          ))}
        </dl>
      </div>
      <p className="text-muted-foreground mt-4 text-xs">{t("settings.storage.databaseNote")}</p>
    </figure>
  )
}

function formatPercentage(fraction: number): string {
  if (fraction > 0 && fraction < 0.01) {
    return "<1%"
  }
  return `${Math.round(fraction * 100)}%`
}
