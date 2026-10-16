import { MiMoBadge } from "./mimo-options"
import { KimiCodeResource } from "./kimi-code-resource"
import { MiMoPlanResource } from "./mimo-resource"
import { MiniMaxResource } from "./minimax-resource"
import { MiniMaxBadge } from "./minimax-options"
import { KimiPlatformBadge } from "./kimi-platform"
import { StepFunBadge } from "./stepfun-options"
import { StepFunPlanResource } from "./stepfun-resource"
import { Tooltip } from "../../components/ui/tooltip"
import { CircleHelp } from "lucide-react"
import { useTranslation } from "react-i18next"
import type { AccountItem } from "../../services/omniproxy"
import { accountDisplayName, formatAccountTime } from "./account-display"
import { AccountItemActions, type AccountItemActionsProps } from "./account-item-actions"
import { AccountStatusBadge } from "./account-status-badge"
import { AccountKeyPreview } from "./account-key-preview"
import { SubscriptionQuota } from "./subscription-quota"
import { AccountBalance } from "./account-balance"
import { GoogleAccountResource } from "../google/google-account-resource"

export function AccountDetailsList({
  accounts,
  busy,
  onToggle,
  onDelete,
  onEdit,
  balanceSupported = false,
  googleSupported = false,
}: {
  accounts: AccountItem[]
  balanceSupported?: boolean
  googleSupported?: boolean
} & Omit<AccountItemActionsProps, "account">) {
  const { t } = useTranslation()
  const hasSubscriptions = accounts.some((account) => account.authMethod === "subscription_login")
  return (
    <div
      role="region"
      aria-label={t("accounts.view.tableLabel")}
      tabIndex={0}
      className="bg-card border-border focus-visible:ring-ring/50 overflow-x-auto rounded-xl border outline-none focus-visible:ring-4"
    >
      <table className="w-full min-w-[1000px] border-collapse text-sm">
        <caption className="sr-only">{t("accounts.view.tableLabel")}</caption>
        <thead className="bg-muted text-muted-foreground border-border border-b text-left text-xs">
          <tr>
            <th scope="col" className="bg-muted sticky left-0 z-20 px-4 py-3 font-medium">
              {t("accounts.list.identity")}
            </th>
            <th scope="col" className="px-4 py-3 font-medium">
              {t("subscription.authentication")}
            </th>
            <th scope="col" className="px-4 py-3 font-medium">
              {t("accounts.detail.status")}
            </th>
            <th scope="col" className="px-4 py-3 font-medium">
              {t(hasSubscriptions ? "subscription.quota.title" : "accounts.detail.balance")}
            </th>
            <th scope="col" className="px-4 py-3 font-medium">
              {t("accounts.detail.created")}
            </th>
            <th scope="col" className="px-4 py-3 font-medium">
              {t("accounts.detail.updated")}
            </th>
            <th
              scope="col"
              className="bg-muted sticky right-0 z-20 px-4 py-3 text-right font-medium"
            >
              {t("accounts.list.actions")}
            </th>
          </tr>
        </thead>
        <tbody className="divide-border divide-y">
          {accounts.map((account, index) => (
            <tr key={account.id} className="hover:bg-muted/30 transition-colors">
              <th
                scope="row"
                className="bg-card sticky left-0 z-10 max-w-64 px-4 py-3 text-left font-medium"
              >
                <div className="flex min-w-0 items-center gap-2.5">
                  <span className="bg-muted text-muted-foreground flex size-6 shrink-0 items-center justify-center rounded-md text-xs tabular-nums">
                    {accounts.length - index}
                  </span>
                  <Tooltip content={accountDisplayName(account)}>
                    <span className="truncate">{accountDisplayName(account)}</span>
                  </Tooltip>
                </div>
                <StepFunBadge account={account} />
                <MiniMaxBadge account={account} />
                <MiMoBadge account={account} />
                <KimiPlatformBadge account={account} />
              </th>
              <td className="px-4 py-3">
                <AccountKeyPreview account={account} />
              </td>
              <td className="px-4 py-3">
                <AccountStatusBadge enabled={account.enabled} />
              </td>
              <td className="px-4 py-3">
                {account.authMethod === "subscription_login" ? (
                  <div className="min-w-72">
                    <SubscriptionQuota account={account} />
                  </div>
                ) : googleSupported || account.providerId === "builtin_google" ? (
                  <div className="max-w-md min-w-72">
                    <GoogleAccountResource
                      account={account}
                      accounts={accounts.filter((item) => item.authMethod === "api_key")}
                    />
                  </div>
                ) : account.kimi?.billingMode === "coding" ? (
                  <KimiCodeResource account={account} />
                ) : account.mimo?.billingMode === "token_plan" ? (
                  <MiMoPlanResource account={account} />
                ) : account.minimax || account.providerId === "builtin_minimax" ? (
                  <MiniMaxResource account={account} />
                ) : account.stepfun?.billingMode === "step_plan" ? (
                  <StepFunPlanResource account={account} />
                ) : balanceSupported ? (
                  <div className="min-w-64">
                    <AccountBalance account={account} />
                  </div>
                ) : (
                  <span className="bg-muted/60 text-muted-foreground inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs whitespace-nowrap">
                    <CircleHelp className="size-3" aria-hidden="true" />
                    {t("accounts.detail.balancePending")}
                  </span>
                )}
              </td>
              <td className="text-muted-foreground px-4 py-3 text-xs whitespace-nowrap">
                {formatAccountTime(account.createdAt)}
              </td>
              <td className="text-muted-foreground px-4 py-3 text-xs whitespace-nowrap">
                {formatAccountTime(account.updatedAt)}
              </td>
              <td className="bg-card sticky right-0 z-10 px-4 py-3">
                <div className="flex justify-end">
                  <AccountItemActions
                    account={account}
                    busy={busy}
                    onToggle={onToggle}
                    onDelete={onDelete}
                    onEdit={onEdit}
                  />
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
