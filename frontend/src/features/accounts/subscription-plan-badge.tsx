import { useTranslation } from "react-i18next"
import { cn } from "../../lib/utils"

const planColors: Record<string, string> = {
  free: "border-slate-200 bg-slate-100 text-slate-700 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200",
  go: "border-teal-200 bg-teal-50 text-teal-800 dark:border-teal-800 dark:bg-teal-950 dark:text-teal-200",
  plus: "border-blue-200 bg-blue-50 text-blue-800 dark:border-blue-800 dark:bg-blue-950 dark:text-blue-200",
  team: "border-violet-200 bg-violet-50 text-violet-800 dark:border-violet-800 dark:bg-violet-950 dark:text-violet-200",
  business:
    "border-indigo-200 bg-indigo-50 text-indigo-800 dark:border-indigo-800 dark:bg-indigo-950 dark:text-indigo-200",
  self_serve_business_prolite:
    "border-fuchsia-200 bg-fuchsia-50 text-fuchsia-800 dark:border-fuchsia-800 dark:bg-fuchsia-950 dark:text-fuchsia-200",
  self_serve_business_usage_based:
    "border-cyan-200 bg-cyan-50 text-cyan-800 dark:border-cyan-800 dark:bg-cyan-950 dark:text-cyan-200",
  pro: "border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200",
  prolite:
    "border-orange-200 bg-orange-50 text-orange-800 dark:border-orange-800 dark:bg-orange-950 dark:text-orange-200",
  enterprise:
    "border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-800 dark:bg-emerald-950 dark:text-emerald-200",
}

export function SubscriptionPlanBadge({ plan }: { plan?: string }) {
  const { t } = useTranslation()
  // 套餐大类无法可靠推断价格或额度档位，保留服务端标识以免显示虚假的等级。
  const value = plan?.trim()
  const color = value ? planColors[value.toLowerCase()] : undefined
  return (
    <span
      className={cn(
        "inline-flex max-w-full items-center rounded-lg border px-2.5 py-1 text-xs font-medium break-all",
        color || "border-border bg-muted text-muted-foreground",
      )}
    >
      {value || t("subscription.planUnknown")}
    </span>
  )
}
