import { RefreshCw } from "lucide-react"
import { useEffect } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import type { AccountItem } from "../../services/omniproxy"
import { subscriptionError } from "../../services/subscription"
import { useResetCredits } from "./use-reset-credits"
import { ResetCreditList } from "./reset-credit-list"

export function ResetCreditsContent({
  account,
  onBusyChange,
}: {
  account: AccountItem
  onBusyChange: (busy: boolean) => void
}) {
  const { t } = useTranslation()
  const wallet = useResetCredits(account)
  const summary = wallet.quota.data?.resetCredits
  const cards = summary?.credits || []
  const busy = wallet.mutation.isPending
  useEffect(() => onBusyChange(busy), [busy, onBusyChange])
  const locked = busy || !!wallet.pending || wallet.status.isPending || wallet.status.isError
  const selected = cards.find((card) => card.id === wallet.selection)
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      <div className="bg-muted/30 flex items-center justify-between rounded-xl border p-4">
        <div>
          <p className="text-muted-foreground text-xs">{t("subscription.reset.count")}</p>
          <p className="mt-1 text-2xl font-semibold tabular-nums">
            {summary ? summary.availableCount : "—"}
          </p>
        </div>
        <Tooltip content={t("subscription.reset.refresh")}>
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={t("subscription.reset.refresh")}
            disabled={busy || wallet.quota.isFetching}
            onClick={() => {
              void wallet.quota.refetch()
              void wallet.status.refetch()
            }}
          >
            <RefreshCw className="size-4" aria-hidden="true" />
          </Button>
        </Tooltip>
      </div>
      <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
        {wallet.quota.isPending ? (
          <p role="status">{t("subscription.quota.loading")}</p>
        ) : wallet.quota.isError ? (
          <p role="alert" className="text-destructive text-sm">
            {t(`subscription.error.${subscriptionError(wallet.quota.error)}`)}
          </p>
        ) : !summary ? (
          <p className="text-muted-foreground text-sm">{t("subscription.reset.unavailable")}</p>
        ) : summary.availableCount === 0 ? (
          <p className="text-muted-foreground text-sm">{t("subscription.reset.empty")}</p>
        ) : (
          <>
            {!summary.detailsKnown || summary.availableCount > cards.length ? (
              <p className="text-muted-foreground text-xs">{t("subscription.reset.partial")}</p>
            ) : null}
            <ResetCreditList cards={cards} busy={locked} onSelect={wallet.setSelection} />
          </>
        )}
        {wallet.status.isError ? (
          <p role="alert" className="text-destructive text-sm">
            {t("subscription.error.reset_record_failed")}
          </p>
        ) : null}
        {wallet.pending ? (
          <div className="space-y-2 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200">
            <p>{t("subscription.reset.pending")}</p>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={busy}
              onClick={() => void wallet.submit(true)}
            >
              {t("subscription.reset.verify")}
            </Button>
          </div>
        ) : null}
        {wallet.receipt && wallet.selection === null ? (
          <div className="space-y-1">
            <p className="text-muted-foreground text-xs">{t("subscription.reset.receipt")}</p>
            <p role="status" className="text-sm">
              {t(`subscription.reset.outcome.${wallet.receipt.outcome}`, {
                defaultValue: t("subscription.reset.pending"),
              })}
            </p>
          </div>
        ) : null}
        {wallet.mutation.data?.refreshFailed ? (
          <p role="alert" className="text-muted-foreground text-xs">
            {t("subscription.reset.refreshFailed")}
          </p>
        ) : null}
        {wallet.mutation.isError ? (
          <p role="alert" className="text-destructive text-sm">
            {t(`subscription.error.${subscriptionError(wallet.mutation.error)}`)}
          </p>
        ) : null}
      </div>
      {wallet.selection !== null ? (
        <div className="bg-muted/30 space-y-3 rounded-xl border p-4">
          <p className="text-sm font-medium">
            {t("subscription.reset.confirm", {
              title: selected?.title || t("subscription.reset.next"),
            })}
          </p>
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={busy}
              onClick={() => wallet.setSelection(null)}
            >
              {t("subscription.reset.cancel")}
            </Button>
            <Button
              type="button"
              size="sm"
              disabled={busy || !!wallet.pending}
              onClick={() => void wallet.submit()}
            >
              {t(busy ? "subscription.reset.using" : "subscription.reset.confirmUse")}
            </Button>
          </div>
        </div>
      ) : (
        <Button
          type="button"
          variant="outline"
          disabled={locked || !summary || summary.availableCount <= 0}
          onClick={() => wallet.setSelection("")}
        >
          {t("subscription.reset.useNext")}
        </Button>
      )}
    </div>
  )
}
