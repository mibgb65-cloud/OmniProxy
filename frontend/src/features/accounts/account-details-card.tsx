import { useTranslation } from "react-i18next"
import { Card, CardContent } from "../../components/ui/card"
import { formatAccountTime } from "./account-display"
import { AccountItemActions, type AccountItemActionsProps } from "./account-item-actions"
import { AccountStatusBadge } from "./account-status-badge"
import { AccountKeyPreview } from "./account-key-preview"

export function AccountDetailsCard({ account, busy, onToggle, onDelete }: AccountItemActionsProps) {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage === "zh" ? "zh-CN" : "en-US"

  return (
    <Card className="dark:bg-card/75 min-w-0 gap-0 py-0 shadow-sm">
      <CardContent className="space-y-4 px-5 py-4">
        <div className="flex min-w-0 flex-wrap items-center justify-between gap-3">
          <div className="min-w-0">
            <h3 className="truncate font-semibold" title={account.name}>
              {account.name}
            </h3>
            <p className="text-muted-foreground text-xs">{t("accounts.apiKey")}</p>
          </div>
          <AccountItemActions
            account={account}
            busy={busy}
            onToggle={onToggle}
            onDelete={onDelete}
          />
        </div>
        <AccountKeyPreview account={account} />
        <dl className="border-border grid grid-cols-2 gap-4 border-t pt-4 text-sm">
          <div>
            <dt className="text-muted-foreground text-xs">{t("accounts.detail.status")}</dt>
            <dd className="mt-1">
              <AccountStatusBadge enabled={account.enabled} />
            </dd>
          </div>
          <div>
            <dt className="text-muted-foreground text-xs">{t("accounts.detail.balance")}</dt>
            <dd>{t("accounts.detail.balancePending")}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground text-xs">{t("accounts.detail.created")}</dt>
            <dd className="mt-1 text-xs break-words">
              {formatAccountTime(account.createdAt, locale)}
            </dd>
          </div>
          <div>
            <dt className="text-muted-foreground text-xs">{t("accounts.detail.updated")}</dt>
            <dd className="mt-1 text-xs break-words">
              {formatAccountTime(account.updatedAt, locale)}
            </dd>
          </div>
        </dl>
      </CardContent>
    </Card>
  )
}
