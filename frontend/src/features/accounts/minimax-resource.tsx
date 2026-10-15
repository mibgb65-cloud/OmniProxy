import { useState } from "react"
import { useTranslation } from "react-i18next"
import { RefreshCw } from "lucide-react"
import { Button } from "../../components/ui/button"
import type { AccountItem } from "../../services/omniproxy"
import {
  miniMaxError,
  openMiniMaxConsole,
  validMiniMaxOptions,
  type MiniMaxQuotaWindow,
} from "../../services/minimax"
import { useMiniMaxQuota } from "./use-minimax-quota"
import { formatAccountTime } from "./account-display"

export function MiniMaxResource({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const query = useMiniMaxQuota(account)
  const [opening, setOpening] = useState(false)
  const [openFailed, setOpenFailed] = useState(false)
  const configured = validMiniMaxOptions(account.minimax)
  const subscription = account.minimax?.billingMode === "subscription"
  async function open() {
    setOpening(true)
    setOpenFailed(false)
    try {
      await openMiniMaxConsole(account.id)
    } catch {
      setOpenFailed(true)
    } finally {
      setOpening(false)
    }
  }
  return (
    <section
      className="bg-muted/20 space-y-3 rounded-xl border p-3"
      aria-label={t("minimax.resources")}
      aria-busy={query.isFetching}
    >
      <div className="flex items-center justify-between gap-2">
        <p className="text-sm font-medium">{t("minimax.resources")}</p>
        {configured && subscription ? (
          <Button
            variant="ghost"
            size="icon-sm"
            className="cursor-pointer"
            disabled={query.isFetching}
            onClick={() => void query.refetch({ cancelRefetch: false })}
            aria-label={t("minimax.refresh")}
          >
            <RefreshCw
              className={
                query.isFetching ? "size-4 animate-spin motion-reduce:animate-none" : "size-4"
              }
              aria-hidden="true"
            />
          </Button>
        ) : null}
      </div>
      {!configured ? (
        <p className="text-muted-foreground text-xs">{t("minimax.needsConfiguration")}</p>
      ) : !subscription ? (
        <p className="text-muted-foreground text-xs leading-5">{t("minimax.balanceUnknown")}</p>
      ) : (
        <>
          {(query.data?.buckets ?? []).map((bucket) => (
            <div key={bucket.model} className="space-y-2 border-t pt-2">
              <p className="truncate text-xs font-medium" title={bucket.model}>
                {bucket.model}
              </p>
              <Window label={t("minimax.interval")} value={bucket.interval} />
              <Window label={t("minimax.weekly")} value={bucket.weekly} />
            </div>
          ))}
          {!query.data ? (
            <p className="text-muted-foreground text-xs">
              {t(query.isPending ? "minimax.loading" : "minimax.unknown")}
            </p>
          ) : null}
          {query.data?.buckets?.length === 0 ? (
            <p className="text-muted-foreground text-xs">{t("minimax.unknown")}</p>
          ) : null}
          {query.isError ? (
            <p role="alert" className="text-destructive text-xs">
              {t(`minimax.error.${miniMaxError(query.error)}`)}
            </p>
          ) : null}
          {query.data ? (
            <p className="text-muted-foreground text-[11px]">
              {t("minimax.fetchedAt", { time: formatAccountTime(query.data.fetchedAt) })}
            </p>
          ) : null}
          <p className="text-muted-foreground text-[11px] leading-5">{t("minimax.resourceHint")}</p>
        </>
      )}
      {configured ? (
        <Button
          size="sm"
          variant="outline"
          className="cursor-pointer"
          disabled={opening}
          onClick={() => void open()}
        >
          {t("minimax.openPlatform")}
        </Button>
      ) : null}
      {openFailed ? (
        <p role="alert" className="text-destructive text-xs">
          {t("minimax.openFailed")}
        </p>
      ) : null}
    </section>
  )
}

function Window({ label, value }: { label: string; value: MiniMaxQuotaWindow }) {
  const { t, i18n } = useTranslation()
  const percent = value.remainingPercent
  return (
    <div className="space-y-1 text-xs">
      <div className="flex flex-wrap justify-between gap-2">
        <span className="text-muted-foreground">{label}</span>
        <span className="font-mono tabular-nums">
          {value.status === "unlimited"
            ? t("minimax.unlimited")
            : percent == null
              ? t("minimax.unknown")
              : `${new Intl.NumberFormat(i18n.language, { maximumFractionDigits: 1 }).format(percent)}%`}
        </span>
      </div>
      {percent != null ? (
        <div
          role="meter"
          aria-label={label}
          aria-valuenow={percent}
          aria-valuemin={0}
          aria-valuemax={100}
          className="bg-muted h-1.5 overflow-hidden rounded-full"
        >
          <div className="h-full rounded-full bg-blue-500" style={{ width: `${percent}%` }} />
        </div>
      ) : null}
      {value.boostPermille != null && value.boostPermille !== 1000 ? (
        <p className="text-muted-foreground text-[11px]">
          {t("minimax.boost", { value: value.boostPermille / 1000 })}
        </p>
      ) : null}
      {value.resetsAt ? (
        <p className="text-muted-foreground text-[11px]">
          {t("minimax.resetsAt", { time: formatAccountTime(value.resetsAt) })}
        </p>
      ) : null}
    </div>
  )
}
