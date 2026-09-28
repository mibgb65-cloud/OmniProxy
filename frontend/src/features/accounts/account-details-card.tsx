import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent } from "../../components/ui/card"
import type { AccountItem } from "../../services/omniproxy"
import { ProviderSwitch } from "../providers/provider-switch"

function formatTime(value: string, locale: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "—"
  return new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }).format(date)
}

export function AccountDetailsCard({
  account,
  busy,
  onToggle,
  onDelete,
}: {
  account: AccountItem
  busy: boolean
  onToggle: (id: string, enabled: boolean) => void
  onDelete: (account: AccountItem) => void
}) {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage === "zh" ? "zh-CN" : "en-US"

  return (
    <Card className="dark:bg-card/75 gap-0 py-0 shadow-sm">
      <CardContent className="space-y-4 px-5 py-4">
        <div className="flex min-w-0 flex-wrap items-center justify-between gap-3">
          <div className="min-w-0">
            <h3 className="truncate font-semibold" title={account.name}>
              {account.name}
            </h3>
            <p className="text-muted-foreground text-xs">{t("accounts.apiKey")}</p>
          </div>
          <div className="flex items-center gap-3">
            <ProviderSwitch
              name={account.name}
              ariaLabel={t("accounts.toggleLabel", { name: account.name })}
              checked={account.enabled}
              disabled={busy}
              onChange={(enabled) => onToggle(account.id, enabled)}
            />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={busy}
              onClick={() => onDelete(account)}
              aria-label={t("accounts.deleteLabel", { name: account.name })}
              className="cursor-pointer"
            >
              {t("accounts.delete")}
            </Button>
          </div>
        </div>
        <dl className="border-border grid gap-4 border-t pt-4 text-sm sm:grid-cols-2 xl:grid-cols-4">
          <div>
            <dt className="text-muted-foreground text-xs">{t("accounts.detail.status")}</dt>
            <dd>{account.enabled ? t("accounts.enabled") : t("accounts.disabled")}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground text-xs">{t("accounts.detail.balance")}</dt>
            <dd>{t("accounts.detail.balancePending")}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground text-xs">{t("accounts.detail.created")}</dt>
            <dd>{formatTime(account.createdAt, locale)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground text-xs">{t("accounts.detail.updated")}</dt>
            <dd>{formatTime(account.updatedAt, locale)}</dd>
          </div>
        </dl>
      </CardContent>
    </Card>
  )
}
