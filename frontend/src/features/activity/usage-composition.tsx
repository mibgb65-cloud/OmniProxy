import { useTranslation } from "react-i18next"
import { chartCompact } from "../../lib/chart"
import { formatTokens, type Summary } from "../../services/activity"
import { Donut, UsageCard, UsageEmpty } from "./usage-charts"
import {
  usageInputOutputRatio,
  usagePercentLabel,
  usageRequestAverage,
  usageShare,
} from "./usage-model"

export function UsageComposition({ summary }: { summary: Summary }) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const input = BigInt(summary.inputTokens)
  const output = BigInt(summary.outputTokens)
  const total = input + output
  const inputShare = usageShare(input, total)
  const outputShare = usageShare(output, total)
  const ratio = usageInputOutputRatio(summary.inputTokens, summary.outputTokens)
  return (
    <UsageCard title={t("activity.composition")} hint={t("activity.compositionHint")}>
      {total === 0n ? (
        <UsageEmpty>{t("activity.empty")}</UsageEmpty>
      ) : (
        <div className="flex flex-1 flex-wrap items-center justify-center gap-5 sm:justify-start">
          <Donut
            label={t("activity.composition")}
            segments={[
              { key: "input", value: input, className: "stroke-blue-500" },
              { key: "output", value: output, className: "stroke-emerald-500" },
            ]}
            center={
              <>
                <span className="text-muted-foreground text-[10px]">Token</span>
                <span className="font-mono text-lg font-semibold tabular-nums">
                  {chartCompact(total, language)}
                </span>
              </>
            }
          />
          <dl className="min-w-0 flex-1 space-y-3">
            {[
              {
                key: "input",
                label: t("activity.input"),
                value: formatTokens(summary.inputTokens, language),
                percent: inputShare,
                className: "bg-blue-500",
              },
              {
                key: "output",
                label: t("activity.output"),
                value: formatTokens(summary.outputTokens, language),
                percent: outputShare,
                className: "bg-emerald-500",
              },
            ].map((item) => (
              <div key={item.key} className="min-w-0">
                <dt className="flex items-center justify-between gap-3 text-xs">
                  <span className="text-muted-foreground inline-flex items-center gap-1.5">
                    <span className={`size-2 rounded-full ${item.className}`} aria-hidden="true" />
                    {item.label}
                  </span>
                  <span className="font-mono tabular-nums">
                    {usagePercentLabel(item.percent, language)}
                  </span>
                </dt>
                <dd className="mt-0.5 font-mono text-sm font-medium tabular-nums">{item.value}</dd>
              </div>
            ))}
          </dl>
        </div>
      )}
      <dl className="mt-4 grid grid-cols-3 gap-2 border-t pt-3">
        <div className="min-w-0">
          <dt className="text-muted-foreground truncate text-[11px]">
            {t("activity.perRequestInput")}
          </dt>
          <dd className="mt-0.5 font-mono text-xs font-medium tabular-nums">
            {formatTokens(
              usageRequestAverage(summary.inputTokens, summary.requests) ?? "0",
              language,
            )}
          </dd>
        </div>
        <div className="min-w-0">
          <dt className="text-muted-foreground truncate text-[11px]">
            {t("activity.perRequestOutput")}
          </dt>
          <dd className="mt-0.5 font-mono text-xs font-medium tabular-nums">
            {formatTokens(
              usageRequestAverage(summary.outputTokens, summary.requests) ?? "0",
              language,
            )}
          </dd>
        </div>
        <div className="min-w-0">
          <dt className="text-muted-foreground truncate text-[11px]">
            {t("activity.inputOutputRatio")}
          </dt>
          <dd className="mt-0.5 font-mono text-xs font-medium tabular-nums">
            {ratio === null ? "—" : `${ratio.toFixed(1)} : 1`}
          </dd>
        </div>
      </dl>
    </UsageCard>
  )
}
