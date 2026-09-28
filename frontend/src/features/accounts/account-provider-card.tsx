import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent } from "../../components/ui/card"
import type { AccountItem, ProviderConfiguration } from "../../services/omniproxy"
import { ProviderBrandIcon } from "../providers/provider-brand-icon"
import { isKnownProviderKind, providerNameKeys } from "../providers/provider-labels"
import { ProviderSwitch } from "../providers/provider-switch"

export function AccountProviderCard({
  provider,
  accounts,
  busy,
  onAdd,
  onToggle,
  onDelete,
}: {
  provider: ProviderConfiguration
  accounts: AccountItem[]
  busy: boolean
  onAdd: () => void
  onToggle: (id: string, enabled: boolean) => void
  onDelete: (account: AccountItem) => void
}) {
  const { t } = useTranslation()
  const name =
    provider.source === "builtin" && isKnownProviderKind(provider.kind)
      ? t(providerNameKeys[provider.kind])
      : provider.name

  return (
    <Card className="dark:bg-card/75 gap-0 py-0 shadow-sm">
      <CardContent className="space-y-4 px-4 py-4">
        <div className="flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <ProviderBrandIcon kind={provider.kind} />
            <div className="min-w-0">
              <h2 className="truncate text-sm font-semibold" title={name}>
                {name}
              </h2>
              <p className="text-muted-foreground text-xs">
                {t("accounts.count", { count: accounts.length })}
              </p>
            </div>
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onAdd}
            className="shrink-0 cursor-pointer"
          >
            {t("accounts.add")}
          </Button>
        </div>
        {!provider.enabled ? (
          <p className="text-muted-foreground text-xs">{t("accounts.providerDisabled")}</p>
        ) : null}
        {provider.kind === "openai" && provider.source === "builtin" ? (
          <p className="text-muted-foreground text-xs">{t("accounts.subscriptionPlanned")}</p>
        ) : null}
        {accounts.length === 0 ? (
          <p className="text-muted-foreground border-border rounded-lg border border-dashed px-3 py-4 text-sm">
            {t("accounts.empty")}
          </p>
        ) : (
          <ul className="divide-border divide-y rounded-lg border">
            {accounts.map((account) => (
              <li
                key={account.id}
                className="flex min-w-0 items-center justify-between gap-3 px-3 py-2.5"
              >
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium" title={account.name}>
                    {account.name}
                  </p>
                  <p className="text-muted-foreground text-xs">
                    {t("accounts.apiKey")} ·{" "}
                    {account.enabled ? t("accounts.enabled") : t("accounts.disabled")}
                  </p>
                </div>
                <div className="flex shrink-0 items-center gap-2">
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
                    className="cursor-pointer"
                    aria-label={t("accounts.deleteLabel", { name: account.name })}
                  >
                    {t("accounts.delete")}
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}
