import { useState } from "react"
import { RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import type { AccountItem } from "../../services/omniproxy"
import { formatAccountTime } from "../accounts/account-display"
import { dumateReasonKey } from "./dumate-error"
import { formatPoints } from "./dumate-format"
import { checkinMessageKey, useDuMateCheckin, useDuMatePoints } from "./use-dumate"

/** 积分只来自公开 DTO；失败与历史快照分开显示，不把失败当零余额。 */
export function DuMatePoints({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const query = useDuMatePoints(account)
  const checkin = useDuMateCheckin(account)
  const [notice, setNotice] = useState("")
  const data = query.data
  const points = data?.points
  const reason = (code?: string) => t(dumateReasonKey(code))

  const runCheckin = () => {
    setNotice("")
    checkin.mutate(undefined, {
      onSuccess: (result) =>
        setNotice(
          t(checkinMessageKey(result), {
            credit: result.credit ? ` +${formatPoints(result.credit)}` : "",
            days: result.totalTimes ?? "?",
            reason: reason(result.errorCode),
          }),
        ),
      onError: () => setNotice(t("dumate.points.checkinFailed", { reason: reason(undefined) })),
    })
  }

  const showCheckinButton = !!points && (!points.checkinKnown || !points.checkinIssued)

  return (
    <div
      aria-busy={query.isFetching}
      className="bg-muted/20 min-w-0 space-y-2.5 rounded-xl border p-3"
    >
      <div className="flex min-w-0 items-center justify-between gap-2">
        <h4 className="text-xs font-medium">{t("dumate.points.title")}</h4>
        <Tooltip content={t("dumate.points.refresh")}>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            aria-label={t("dumate.points.refresh")}
            disabled={query.isFetching}
            onClick={() => void query.refetch()}
            className="cursor-pointer"
          >
            <RefreshCw
              className={`size-3 ${query.isFetching ? "animate-spin motion-reduce:animate-none" : ""}`}
              aria-hidden="true"
            />
          </Button>
        </Tooltip>
      </div>
      {query.isPending ? (
        <p role="status" className="text-muted-foreground text-xs">
          {t("dumate.points.loading")}
        </p>
      ) : query.isError || data?.state === "unavailable" ? (
        <p role="alert" className="text-destructive text-xs">
          {t("dumate.points.unavailable", {
            reason: reason(data?.errorCode ?? "operation_failed"),
          })}
        </p>
      ) : points ? (
        <>
          <div className="grid grid-cols-2 gap-2 text-xs">
            <div>
              <p className="text-muted-foreground">{t("dumate.points.remaining")}</p>
              <p className="mt-0.5 font-mono text-sm tabular-nums">
                {t("dumate.points.remainingValue", {
                  points: formatPoints(points.remainingPoints),
                })}
              </p>
            </div>
            <div>
              <p className="text-muted-foreground">{t("dumate.points.used")}</p>
              <p className="mt-0.5 font-mono text-sm tabular-nums">
                {t("dumate.points.usage", {
                  used: formatPoints(points.usedPoints),
                  total: formatPoints(points.totalPoints),
                })}
              </p>
            </div>
          </div>
          {points.expiringAt ? (
            <p className="text-muted-foreground text-[11px]">
              {t("dumate.points.expiring", {
                points: formatPoints(points.expiringPoints ?? 0),
                time: formatAccountTime(points.expiringAt),
              })}
            </p>
          ) : null}
          <div className="flex flex-wrap items-center justify-between gap-2 border-t pt-2 text-xs">
            <span className="text-muted-foreground">
              {points.checkinKnown
                ? t(points.checkinIssued ? "dumate.points.checkin" : "dumate.points.checkinOpen", {
                    days: points.checkinDays,
                  })
                : t("dumate.points.checkinUnknown")}
            </span>
            {showCheckinButton ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={checkin.isPending}
                onClick={runCheckin}
                className="h-7 cursor-pointer px-2 text-xs"
              >
                {t(checkin.isPending ? "dumate.points.checking" : "dumate.points.checkinButton")}
              </Button>
            ) : null}
          </div>
          {notice ? (
            <p role="status" aria-live="polite" className="text-muted-foreground text-[11px]">
              {notice}
            </p>
          ) : null}
          {data?.state === "stale" ? (
            <p className="text-destructive text-[11px]">
              {t("dumate.points.stale", {
                time: data.fetchedAt ? formatAccountTime(data.fetchedAt) : "—",
                reason: reason(data.errorCode),
              })}
            </p>
          ) : (
            <p className="text-muted-foreground text-[10px]">
              {data?.fetchedAt
                ? t("dumate.points.fetchedAt", { time: formatAccountTime(data.fetchedAt) })
                : t("dumate.points.never")}
            </p>
          )}
        </>
      ) : null}
    </div>
  )
}
