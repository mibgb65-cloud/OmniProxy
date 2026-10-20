import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { LogOut, RefreshCw } from "lucide-react"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import { StatusToast } from "../../components/common/status-toast"
import type { AccountItem } from "../../services/omniproxy"
import {
  logoutSubscription,
  refreshSubscription,
  subscriptionError,
  subscriptionRefreshError,
  transientSubscriptionRefreshError,
} from "../../services/subscription"
import { SubscriptionLoginDialog } from "./subscription-login-dialog"
import { SubscriptionPlanBadge } from "./subscription-plan-badge"
import { SubscriptionAccountID } from "./subscription-account-id"
import { ResetCreditsPopover } from "./reset-credits-popover"

export function SubscriptionAccountStatus({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const mutation = useMutation({
    mutationFn: ({ action, id }: { action: "refresh" | "logout"; id: string; version: string }) =>
      action === "refresh" ? refreshSubscription(id) : logoutSubscription(id),
    onSettled: () => client.invalidateQueries({ queryKey: ["accounts"] }),
  })
  const refreshing =
    mutation.variables?.action === "refresh" && account.providerId !== "builtin_google"
  const errorCode = refreshing
    ? subscriptionRefreshError(mutation.error)
    : subscriptionError(mutation.error)
  // 后续重新登录或账号更新后，旧操作的失败不再代表当前账号。
  const currentError =
    mutation.isError &&
    mutation.variables?.id === account.id &&
    mutation.variables.version === account.updatedAt
  const transient = refreshing && transientSubscriptionRefreshError(errorCode)
  return (
    <div className="min-w-0 space-y-2 text-xs">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-2 gap-y-1.5">
          <SubscriptionPlanBadge plan={account.authPlan} />
          <SubscriptionAccountID id={account.authAccountId} />
        </div>
        <div className="text-muted-foreground flex shrink-0 items-center gap-1">
          {account.providerId !== "builtin_google" ? (
            <ResetCreditsPopover account={account} disabled={mutation.isPending} />
          ) : null}
          <Tooltip content={t("subscription.refresh")}>
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              disabled={mutation.isPending}
              onClick={() =>
                mutation.mutate({ action: "refresh", id: account.id, version: account.updatedAt })
              }
              aria-label={t("subscription.refresh")}
              className="hover:text-foreground cursor-pointer"
            >
              <RefreshCw
                className={
                  mutation.isPending && mutation.variables?.action === "refresh"
                    ? "size-4 animate-spin motion-reduce:animate-none"
                    : "size-4"
                }
                aria-hidden="true"
              />
            </Button>
          </Tooltip>
          <SubscriptionLoginDialog
            providerId={account.providerId}
            accountId={account.id}
            disabled={mutation.isPending}
          />
          {account.authState !== "logged_out" ? (
            <Tooltip content={t("subscription.logout")}>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                disabled={mutation.isPending}
                onClick={() =>
                  mutation.mutate({ action: "logout", id: account.id, version: account.updatedAt })
                }
                aria-label={t("subscription.logout")}
                className="hover:bg-destructive/10 hover:text-destructive cursor-pointer"
              >
                <LogOut className="size-4" aria-hidden="true" />
              </Button>
            </Tooltip>
          ) : null}
        </div>
      </div>
      {account.authState !== "authenticated" ? (
        <p className="text-muted-foreground">
          {t(`subscription.state.${account.authState || "unknown"}`, {
            defaultValue: t("subscription.state.unknown"),
          })}
        </p>
      ) : null}
      {currentError && transient ? (
        <StatusToast
          key={mutation.submittedAt}
          tone="warning"
          title={t(`subscription.error.${errorCode}`)}
          closeLabel={t("subscription.refreshDismiss")}
        />
      ) : currentError ? (
        <p role="alert" className="text-destructive">
          {t(`subscription.error.${errorCode}`)}
        </p>
      ) : null}
    </div>
  )
}
