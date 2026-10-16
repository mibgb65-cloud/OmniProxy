import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { LogOut, RefreshCw } from "lucide-react"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import type { AccountItem } from "../../services/omniproxy"
import {
  logoutSubscription,
  refreshSubscription,
  subscriptionError,
} from "../../services/subscription"
import { SubscriptionLoginDialog } from "./subscription-login-dialog"
import { SubscriptionPlanBadge } from "./subscription-plan-badge"
import { SubscriptionAccountID } from "./subscription-account-id"
import { ResetCreditsPopover } from "./reset-credits-popover"

export function SubscriptionAccountStatus({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const mutation = useMutation({
    mutationFn: (action: "refresh" | "logout") =>
      action === "refresh" ? refreshSubscription(account.id) : logoutSubscription(account.id),
    onSettled: () => client.invalidateQueries({ queryKey: ["accounts"] }),
  })
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
              onClick={() => mutation.mutate("refresh")}
              aria-label={t("subscription.refresh")}
              className="hover:text-foreground cursor-pointer"
            >
              <RefreshCw
                className={
                  mutation.isPending && mutation.variables === "refresh"
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
                onClick={() => mutation.mutate("logout")}
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
      {mutation.isError ? (
        <p role="alert" className="text-destructive">
          {t(`subscription.error.${subscriptionError(mutation.error)}`)}
        </p>
      ) : null}
    </div>
  )
}
