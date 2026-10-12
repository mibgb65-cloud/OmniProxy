import { Tooltip } from "../../components/ui/tooltip"
import { useEffect } from "react"
import { Check, Copy, LoaderCircle, RotateCcw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { AccountItem } from "../../services/omniproxy"
import { useAccountKeyPreview, useCopyAccountKey } from "./use-account-key"
import { SubscriptionAccountStatus } from "./subscription-account-status"

export function AccountKeyPreview({ account }: { account: AccountItem }) {
  return account.authMethod === "subscription_login" ? (
    <SubscriptionAccountStatus account={account} />
  ) : (
    <APIKeyPreview account={account} />
  )
}

function APIKeyPreview({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const preview = useAccountKeyPreview(account.id)
  const copy = useCopyAccountKey(account.id)
  const { isSuccess, reset } = copy
  useEffect(() => {
    if (!isSuccess) return
    const timer = window.setTimeout(reset, 2000)
    return () => window.clearTimeout(timer)
  }, [isSuccess, reset])
  const copyLabel = copy.isSuccess
    ? t("accounts.key.copied")
    : t("accounts.key.copy", { name: account.name })
  return (
    <div className="flex min-w-0 flex-col items-start gap-1">
      <div className="bg-muted/60 inline-flex max-w-full items-center gap-1 rounded-md px-2 py-0.5">
        <code
          className="text-muted-foreground truncate text-xs"
          aria-label={t("accounts.list.apiKey")}
        >
          {preview.isPending
            ? t("accounts.key.loading")
            : preview.isError
              ? t("accounts.key.unavailable")
              : preview.data}
        </code>
        {preview.isError ? (
          <Tooltip content={t("accounts.key.retry")}>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label={t("accounts.key.retry")}
              disabled={preview.isFetching}
              onClick={() => {
                void preview.refetch()
              }}
              className="shrink-0 cursor-pointer"
            >
              <RotateCcw className="size-3" aria-hidden="true" />
            </Button>
          </Tooltip>
        ) : (
          <Tooltip content={copyLabel}>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label={copyLabel}
              disabled={!preview.data || copy.isPending}
              onClick={() => copy.mutate()}
              className="text-muted-foreground shrink-0 cursor-pointer"
            >
              {copy.isPending ? (
                <LoaderCircle
                  className="size-3 animate-spin motion-reduce:animate-none"
                  aria-hidden="true"
                />
              ) : copy.isSuccess ? (
                <Check
                  className="size-3 text-emerald-600 dark:text-emerald-400"
                  aria-hidden="true"
                />
              ) : (
                <Copy className="size-3" aria-hidden="true" />
              )}
            </Button>
          </Tooltip>
        )}
      </div>
      {copy.isError ? (
        <p role="alert" className="text-destructive text-xs">
          {t("accounts.key.copyFailed")}
        </p>
      ) : null}
    </div>
  )
}
