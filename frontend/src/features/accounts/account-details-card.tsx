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

export function AccountDetailsCard({
  account,
  busy,
  onToggle,
  onDelete,
  onEdit,
  balanceSupported = false,
}: AccountItemActionsProps & { balanceSupported?: boolean }) {
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
            <p className="text-muted-foreground text-xs">
              {t(
                account.authMethod === "subscription_login"
                  ? "subscription.type"
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
        ) : balanceSupported ? (
          <>
            <AccountBalance account={account} />
            <AccountTimeFooter account={account} />
          </>
        ) : (
          <dl className="border-border grid grid-cols-2 gap-4 border-t pt-4 text-sm">
            <div className="col-span-2">
              <dt className="text-muted-foreground text-xs">{t("accounts.detail.balance")}</dt>
              <dd>{t("accounts.detail.balancePending")}</dd>
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
