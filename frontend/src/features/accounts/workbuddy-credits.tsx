import { RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import { cn } from "../../lib/utils"
import type { AccountItem } from "../../services/omniproxy"
import { workbuddyError, type WorkBuddyCreditWindow } from "../../services/workbuddy"
import { useWorkBuddyQuota } from "./use-workbuddy-quota"
import { formatAccountTime } from "./account-display"
import "./subscription-quota.css"

// 上游只输出这四个窗口名；未知名称原样展示，不猜测含义。
const windowNameKeys = {
  Credits: "quota.workbuddy.credits",
  "Extra credits": "quota.workbuddy.extraCredits",
  "Total credits": "quota.workbuddy.totalCredits",
  "Prepaid credits": "quota.workbuddy.prepaid",
} as const

export function WorkBuddyCredits({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const query = useWorkBuddyQuota(account)
  const data = query.data
  const windows = data?.windows ?? []
  // 国内版与国际版共用同一套窗口口径，只有卡片标题区分构建。
  const china = account.providerId === "builtin_workbuddy"
  const titleKey = china ? "quota.workbuddyCn.title" : "quota.workbuddy.title"
  const refreshKey = china ? "quota.workbuddyCn.refresh" : "quota.workbuddy.refresh"
  return (
    <section
      aria-label={t(titleKey)}
      aria-busy={query.isFetching}
      className="bg-muted/20 min-w-0 space-y-3 rounded-xl border p-3"
    >
      <div className="flex items-center justify-between gap-2">
        <span className="flex min-w-0 items-baseline gap-1.5">
          <span className="text-sm font-medium">{t(titleKey)}</span>
          {/* 套餐名只有计费接口给得准（登录身份没有该字段）；未知时不显示。 */}
          {data?.plan ? (
            <span className="text-muted-foreground truncate text-xs">{data.plan}</span>
          ) : null}
        </span>
        <Tooltip content={t(refreshKey)}>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            aria-label={t(refreshKey)}
            disabled={query.isFetching}
            className="text-muted-foreground cursor-pointer"
            onClick={() => void query.refetch({ cancelRefetch: false })}
          >
            <RefreshCw
              className={cn(
                "size-3",
                query.isFetching && "animate-spin motion-reduce:animate-none",
              )}
              aria-hidden="true"
            />
          </Button>
        </Tooltip>
      </div>
      {windows.length > 0 ? (
        <WorkBuddyCreditWindows windows={windows} />
      ) : data || !query.isPending ? (
        <p className="text-muted-foreground text-xs">{t("quota.workbuddy.unknown")}</p>
      ) : (
        <div aria-hidden="true" className="space-y-2">
          <div className="bg-muted h-4 w-32 max-w-full rounded" />
          <div className="bg-muted h-1.5 w-full rounded-full" />
          <div className="bg-muted h-4 w-24 max-w-full rounded" />
        </div>
      )}
      {query.isError ? (
        <p role="alert" className="text-destructive text-xs">
          {t(`workbuddy.error.${workbuddyError(query.error)}`)}
        </p>
      ) : null}
      {data?.fromCache ? (
        <p className="text-muted-foreground text-[10px]">{t("subscription.quota.retained")}</p>
      ) : null}
      {data ? (
        <p className="text-muted-foreground text-[10px]">
          {t("quota.workbuddy.refreshed")} {formatAccountTime(data.fetchedAt)}
        </p>
      ) : null}
    </section>
  )
}

/** 只读展示：配额总览已经有额度快照，这里不再发起第二次查询。 */
export function WorkBuddyCreditWindows({ windows }: { windows: WorkBuddyCreditWindow[] }) {
  return (
    <div className="min-w-0 space-y-2.5">
      {windows.map((window) => (
        <WorkBuddyCreditWindowRow key={window.name} window={window} />
      ))}
    </div>
  )
}

export function WorkBuddyCreditWindowRow({ window }: { window: WorkBuddyCreditWindow }) {
  const { t, i18n } = useTranslation()
  const unknown = t("quota.workbuddy.unknown")
  const labelKey = windowNameKeys[window.name as keyof typeof windowNameKeys]
  const label = labelKey ? t(labelKey) : window.name
  const total = window.name === "Total credits"
  const amount = finite(window.amount)
  const limit = finite(window.limit)
  const used = finite(window.used)
  // 预付余额只有余额没有总量：不参与百分比，也不把缺省值当成 0。
  const progress =
    !window.prepaid && limit !== undefined && limit > 0 && used !== undefined
      ? Math.min(100, Math.max(0, used))
      : undefined
  const amountText = window.prepaid
    ? amount === undefined
      ? unknown
      : formatCredit(amount, i18n.language)
    : amount === undefined && limit === undefined
      ? unknown
      : `${amount === undefined ? unknown : formatCredit(amount, i18n.language)} / ${
          limit === undefined ? unknown : formatCredit(limit, i18n.language)
        }`
  return (
    <div className={cn("min-w-0 space-y-1.5", total && "border-border/70 border-t pt-2.5")}>
      <div className="flex min-w-0 flex-wrap items-baseline justify-between gap-x-2 gap-y-1">
        <span className={cn("min-w-0 truncate text-xs", total ? "font-semibold" : "font-medium")}>
          {label}
        </span>
        <span
          className={cn(
            "font-mono text-[11px] tabular-nums",
            total && "font-semibold",
            amount === undefined && limit === undefined && "text-muted-foreground",
          )}
        >
          {amountText}
        </span>
      </div>
      {progress !== undefined ? (
        <div
          role="progressbar"
          aria-label={`${t("quota.workbuddy.used")} ${label}`}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={Math.round(progress)}
          className="bg-muted h-1.5 overflow-hidden rounded-full"
        >
          <div
            className={cn(
              "h-full rounded-full transition-[width] duration-500 ease-out motion-reduce:transition-none",
              progress >= 100 ? "bg-amber-500" : "bg-blue-600",
            )}
            style={{ width: `${progress}%` }}
          />
        </div>
      ) : null}
      {progress !== undefined && used !== undefined ? (
        <p className="text-muted-foreground text-[10px]">
          {t("quota.workbuddy.used")} {formatPercent(used, i18n.language)}
        </p>
      ) : null}
      {window.resetsAt || window.runsOutAt ? (
        <dl className="text-muted-foreground grid grid-cols-2 gap-x-3 text-[10px] leading-4">
          {window.resetsAt ? (
            <div className="min-w-0">
              <dt className="truncate">{t("quota.workbuddy.resetsAt")}</dt>
              <dd className="font-mono break-words tabular-nums">
                {formatAccountTime(window.resetsAt)}
              </dd>
            </div>
          ) : null}
          {window.runsOutAt ? (
            <div className="min-w-0">
              <dt className="truncate">{t("quota.workbuddy.runsOutAt")}</dt>
              <dd className="font-mono break-words tabular-nums">
                {formatAccountTime(window.runsOutAt)}
              </dd>
            </div>
          ) : null}
        </dl>
      ) : null}
      {window.aside ? (
        <p className="text-muted-foreground text-[10px] leading-4">{t("quota.workbuddy.aside")}</p>
      ) : null}
    </div>
  )
}

// 上游省略未知字段；只接受有限数字，绝不把缺省或非法值显示成 0。
function finite(value: number | undefined): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined
}

function formatCredit(value: number, language: string) {
  return new Intl.NumberFormat(language, { maximumFractionDigits: 2 }).format(value)
}

function formatPercent(value: number, language: string) {
  return `${new Intl.NumberFormat(language, { maximumFractionDigits: 1 }).format(value)}%`
}
