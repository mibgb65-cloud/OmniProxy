import { useTranslation } from "react-i18next"
import type { AccountTokenUsage } from "../../../bindings/omniproxy/internal/application/activity"
import { formatTokens } from "../../services/activity"

function compactTokens(value: string) {
  if (value.length > 20 || !/^\d+$/.test(value)) return "—"
  // 保持十进制整数的精度，不先转换成可能丢精度的 JavaScript Number。
  return new Intl.NumberFormat("en-US", {
    notation: "compact",
    maximumFractionDigits: 2,
  }).format(BigInt(value))
}

export function AccountTokenSummary({
  usage,
  error,
}: {
  usage?: AccountTokenUsage | null
  error: boolean
}) {
  const { t, i18n } = useTranslation()
  const partialCache = !!usage?.unknownCacheRequests && !!usage.cachedInputTokens
  const fields = [
    { label: t("subscription.tokens.input"), value: usage?.inputTokens },
    { label: t("subscription.tokens.output"), value: usage?.outputTokens },
    {
      label: t("subscription.tokens.cached"),
      value: usage?.cachedInputTokens,
      partial: partialCache,
    },
  ]
  return (
    <section className="space-y-2 border-t pt-2.5" aria-label={t("subscription.tokens.title")}>
      <p className="text-xs font-medium">{t("subscription.tokens.title")}</p>
      <dl className="grid min-w-0 grid-cols-3 gap-2 text-xs">
        {fields.map(({ label, value, partial }) => (
          <div key={label} className="bg-muted/40 min-w-0 space-y-1 rounded-lg px-2 py-1.5">
            <dt className="text-muted-foreground">{label}</dt>
            <dd
              className="font-mono break-all tabular-nums"
              title={!error && value ? formatTokens(value, i18n.language) : undefined}
            >
              {!error && value ? `${partial ? "≥ " : ""}${compactTokens(value)}` : "—"}
            </dd>
          </div>
        ))}
      </dl>
    </section>
  )
}
