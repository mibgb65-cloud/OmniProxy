import { Link } from "@tanstack/react-router"
import { ChevronRight, Trash2 } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent } from "../../components/ui/card"
import { ScrollRegion } from "../../components/common/scroll-region"
import type {
  AccountItem,
  CreateAccountInput,
  ProviderConfiguration,
} from "../../services/omniproxy"
import { ProviderBrandIcon } from "../providers/provider-brand-icon"
import { isKnownProviderKind, providerNameKeys } from "../providers/provider-labels"
import { ProviderSwitch } from "../providers/provider-switch"
import { AddAccountDialog } from "./add-account-dialog"

export function AccountProviderCard({
  provider,
  accounts,
  busy,
  onCreate,
  onToggle,
  onDelete,
}: {
  provider: ProviderConfiguration
  accounts: AccountItem[]
  busy: boolean
  onCreate: (input: CreateAccountInput) => Promise<void>
  onToggle: (id: string, enabled: boolean) => void
  onDelete: (account: AccountItem) => void
}) {
  const { t } = useTranslation()
  const name =
    provider.source === "builtin" && isKnownProviderKind(provider.kind)
      ? t(providerNameKeys[provider.kind])
      : provider.name

  return (
    <Card className="dark:bg-card/75 h-[320px] min-w-0 gap-0 py-0 shadow-sm">
      <CardContent className="flex min-h-0 flex-1 flex-col gap-4 px-4 py-4">
        <div className="flex shrink-0 items-center justify-between gap-3">
          <Link
            to="/accounts/$providerId"
            params={{ providerId: provider.id }}
            aria-label={t("accounts.openDetails", { name })}
            className="group focus-visible:ring-ring/50 flex min-w-0 items-center gap-3 rounded-md focus-visible:ring-4"
          >
            <ProviderBrandIcon kind={provider.kind} />
            <div className="min-w-0">
              <h2 className="group-hover:text-primary truncate text-sm font-semibold" title={name}>
                {name}
              </h2>
              <p className="text-muted-foreground text-xs">
                {t("accounts.count", { count: accounts.length })}
              </p>
            </div>
            <ChevronRight className="text-muted-foreground size-4 shrink-0" aria-hidden="true" />
          </Link>
          <AddAccountDialog provider={provider} onCreate={onCreate} />
        </div>
        {!provider.enabled ? (
          <p className="text-muted-foreground shrink-0 text-xs">{t("accounts.providerDisabled")}</p>
        ) : null}
        {accounts.length === 0 ? (
          <p className="text-muted-foreground border-border flex min-h-0 flex-1 items-center justify-center rounded-lg border border-dashed px-3 py-4 text-sm">
            {t("accounts.empty")}
          </p>
        ) : (
          <ScrollRegion
            as="section"
            className="min-h-0 flex-1 rounded-lg border"
            ariaLabel={t("accounts.listFor", { name })}
          >
            <ul className="divide-border divide-y">
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
                      size="icon-sm"
                      disabled={busy}
                      onClick={() => onDelete(account)}
                      className="text-muted-foreground hover:bg-destructive/10 hover:text-destructive cursor-pointer"
                      aria-label={t("accounts.deleteLabel", { name: account.name })}
                      title={t("accounts.delete")}
                    >
                      <Trash2 className="size-4" aria-hidden="true" />
                    </Button>
                  </div>
                </li>
              ))}
            </ul>
          </ScrollRegion>
        )}
      </CardContent>
    </Card>
  )
}
