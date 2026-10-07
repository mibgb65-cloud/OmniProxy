import { CircleHelp } from "lucide-react"
import { useTranslation } from "react-i18next"
import type { AccountItem } from "../../services/omniproxy"
import { formatAccountTime } from "./account-display"
import { AccountItemActions, type AccountItemActionsProps } from "./account-item-actions"
import { AccountStatusBadge } from "./account-status-badge"
import { AccountKeyPreview } from "./account-key-preview"

export function AccountDetailsList({
  accounts,
  busy,
  onToggle,
  onDelete,
}: {
  accounts: AccountItem[]
} & Omit<AccountItemActionsProps, "account">) {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage === "zh" ? "zh-CN" : "en-US"
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
              {t("accounts.list.apiKey")}
            </th>
            <th scope="col" className="px-4 py-3 font-medium">
              {t("accounts.detail.status")}
            </th>
            <th scope="col" className="px-4 py-3 font-medium">
              {t("accounts.detail.balance")}
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
                  <span className="truncate" title={account.name}>
                    {account.name}
                  </span>
                </div>
              </th>
              <td className="px-4 py-3">
                <AccountKeyPreview account={account} />
              </td>
              <td className="px-4 py-3">
                <AccountStatusBadge enabled={account.enabled} />
              </td>
              <td className="px-4 py-3">
                <span className="bg-muted/60 text-muted-foreground inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs whitespace-nowrap">
                  <CircleHelp className="size-3" aria-hidden="true" />
                  {t("accounts.detail.balancePending")}
                </span>
              </td>
              <td className="text-muted-foreground px-4 py-3 text-xs whitespace-nowrap">
                {formatAccountTime(account.createdAt, locale)}
              </td>
              <td className="text-muted-foreground px-4 py-3 text-xs whitespace-nowrap">
                {formatAccountTime(account.updatedAt, locale)}
              </td>
              <td className="bg-card sticky right-0 z-10 px-4 py-3">
                <div className="flex justify-end">
                  <AccountItemActions
                    account={account}
                    busy={busy}
                    onToggle={onToggle}
                    onDelete={onDelete}
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
