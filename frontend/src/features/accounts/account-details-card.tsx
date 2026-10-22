import { MiMoBadge } from "./mimo-options"
import { KimiCodeResource } from "./kimi-code-resource"
import { MiMoPlanResource } from "./mimo-resource"
import { MiniMaxResource } from "./minimax-resource"
import { MiniMaxBadge } from "./minimax-options"
import { KimiPlatformBadge } from "./kimi-platform"
import { StepFunBadge } from "./stepfun-options"
import { StepFunPlanResource } from "./stepfun-resource"
import { Tooltip } from "../../components/ui/tooltip"
import { useTranslation } from "react-i18next"
import { Card, CardContent } from "../../components/ui/card"
import { accountDisplayName, formatAccountTime } from "./account-display"
import { AccountItemActions, type AccountItemActionsProps } from "./account-item-actions"
import { AccountStatusBadge } from "./account-status-badge"
import { AccountKeyPreview } from "./account-key-preview"
import { SubscriptionQuota } from "./subscription-quota"
import { AccountTimeFooter } from "./account-time-footer"
import { AccountBalance } from "./account-balance"
import { subscriptionTypeKey } from "../providers/provider-labels"
import type { AccountItem } from "../../services/omniproxy"
import { GoogleAccountResource } from "../google/google-account-resource"
import { AccountUsagePanel } from "./account-usage-panel"

export function AccountDetailsCard({
  account,
  busy,
  onToggle,
  onDelete,
  onEdit,
  balanceSupported = false,
  googleSupported = false,
  googleAccounts,
}: AccountItemActionsProps & {
  balanceSupported?: boolean
  googleSupported?: boolean
  googleAccounts?: AccountItem[]
}) {
  const { t } = useTranslation()
  const subscription = account.authMethod === "subscription_login"

  return (
    <Card className="dark:bg-card/75 min-w-0 gap-0 py-0 shadow-sm">
      <CardContent className="flex flex-1 flex-col gap-4 px-5 py-4">
        <div className="flex min-w-0 items-start justify-between gap-3">
          <div className="min-w-0 flex-1 space-y-1">
            <Tooltip content={accountDisplayName(account)}>
              <h3 className="truncate font-semibold">{accountDisplayName(account)}</h3>
            </Tooltip>
            <StepFunBadge account={account} />
            <MiniMaxBadge account={account} />
            <MiMoBadge account={account} />
            <KimiPlatformBadge account={account} />
            <p className="text-muted-foreground text-xs">
              {t(
                account.authMethod === "subscription_login"
                  ? subscriptionTypeKey(account.providerId)
                  : "accounts.apiKey",
              )}
            </p>
          </div>
          <AccountItemActions
            account={account}
            busy={busy}
            onToggle={onToggle}
            onDelete={onDelete}
            onEdit={onEdit}
          />
        </div>
        {subscription ? (
          <AccountKeyPreview account={account} />
        ) : (
          <div className="flex min-w-0 items-center justify-between gap-3">
            <div className="min-w-0 flex-1">
              <AccountKeyPreview account={account} />
            </div>
            <div className="flex shrink-0 items-center">
              <AccountStatusBadge enabled={account.enabled} />
            </div>
          </div>
        )}
        {subscription ? (
          <>
            <SubscriptionQuota account={account} />
            <AccountTimeFooter account={account} />
          </>
        ) : account.providerId === "builtin_river" ? (
          <>
            <AccountUsagePanel account={account} />
            <AccountTimeFooter account={account} />
          </>
        ) : googleSupported || account.providerId === "builtin_google" ? (
          <>
            <GoogleAccountResource account={account} accounts={googleAccounts ?? [account]} />
            <AccountTimeFooter account={account} />
          </>
        ) : account.kimi?.billingMode === "coding" ? (
          <>
            <KimiCodeResource account={account} />
            <AccountTimeFooter account={account} />
          </>
        ) : account.mimo?.billingMode === "token_plan" ? (
          <>
            <MiMoPlanResource account={account} />
            <AccountTimeFooter account={account} />
          </>
        ) : account.minimax || account.providerId === "builtin_minimax" ? (
          <>
            <MiniMaxResource account={account} />
            <AccountTimeFooter account={account} />
          </>
        ) : account.stepfun?.billingMode === "step_plan" ? (
          <>
            <StepFunPlanResource account={account} />
            <AccountTimeFooter account={account} />
          </>
        ) : balanceSupported ? (
          <>
            <AccountBalance account={account} />
            <AccountTimeFooter account={account} />
          </>
        ) : (
          <dl className="border-border grid grid-cols-2 gap-4 border-t pt-4 text-sm">
            <div className="col-span-2">
              <dt className="text-muted-foreground text-xs">{t("accounts.detail.balance")}</dt>
              <dd>{t(googleSupported ? "google.unknown" : "accounts.detail.balancePending")}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground text-xs">{t("accounts.detail.created")}</dt>
              <dd className="mt-1 text-xs break-words">{formatAccountTime(account.createdAt)}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground text-xs">{t("accounts.detail.updated")}</dt>
              <dd className="mt-1 text-xs break-words">{formatAccountTime(account.updatedAt)}</dd>
            </div>
          </dl>
        )}
      </CardContent>
    </Card>
  )
}
