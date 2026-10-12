import { useTranslation } from "react-i18next"
import type { AccountItem } from "../../services/omniproxy"
import { formatAccountTime } from "./account-display"

export function AccountTimeFooter({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  return (
    <dl className="text-muted-foreground mt-auto flex flex-wrap items-center justify-between gap-x-3 gap-y-1 text-[10px]">
      <div className="inline-flex items-baseline gap-1">
        <dt>{t("subscription.quota.created")}</dt>
        <dd>
          <time dateTime={account.createdAt} className="font-mono tabular-nums">
            {formatAccountTime(account.createdAt)}
          </time>
        </dd>
      </div>
      <div className="inline-flex items-baseline gap-1">
        <dt>{t("subscription.quota.updated")}</dt>
        <dd>
          <time dateTime={account.updatedAt} className="font-mono tabular-nums">
            {formatAccountTime(account.updatedAt)}
          </time>
        </dd>
      </div>
    </dl>
  )
}
