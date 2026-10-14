import { StepFunBadge } from "./stepfun-options"
import { Tooltip } from "../../components/ui/tooltip"
import { Link } from "@tanstack/react-router"
import { ChevronRight } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card, CardContent } from "../../components/ui/card"
import { ScrollRegion } from "../../components/common/scroll-region"
import type {
  AccountItem,
  CreateAccountInput,
  CreateAccountsInput,
  ProviderConfiguration,
} from "../../services/omniproxy"
import { ProviderBrandIcon } from "../providers/provider-brand-icon"
import { isKnownProviderKind, providerNameKeys } from "../providers/provider-labels"
import { AccountItemActions } from "./account-item-actions"
import { AddAccountDialog } from "./add-account-dialog"
import { accountDisplayName } from "./account-display"

export function AccountProviderCard({
  provider,
  accounts,
  busy,
  onCreate,
  onBatchCreate,
  onToggle,
  onDelete,
  onEdit,
}: {
  provider: ProviderConfiguration
  accounts: AccountItem[]
  busy: boolean
  onCreate: (input: CreateAccountInput) => Promise<void>
  onBatchCreate: (input: CreateAccountsInput) => Promise<void>
  onToggle: (id: string, enabled: boolean) => void
  onDelete: (account: AccountItem) => void
  onEdit: (account: AccountItem) => void
}) {
  const { t } = useTranslation()
  const name =
    provider.source === "builtin" && isKnownProviderKind(provider.kind)
      ? t(providerNameKeys[provider.kind])
      : provider.name

  return (
    <Card className="dark:bg-card/75 h-[320px] min-w-0 gap-0 py-0 shadow-sm">
      <CardContent className="flex min-h-0 flex-1 flex-col gap-4 px-4 py-4">
        <div className="flex shrink-0 flex-wrap items-center justify-between gap-3">
          <Link
            to="/accounts/$providerId"
            params={{ providerId: provider.id }}
            aria-label={t("accounts.openDetails", { name })}
            className="group focus-visible:ring-ring/50 flex min-w-0 items-center gap-3 rounded-md focus-visible:ring-4"
          >
            <ProviderBrandIcon kind={provider.kind} endpoint={provider.baseUrl} />
            <div className="min-w-0">
              <Tooltip content={name}>
                <h2 className="group-hover:text-primary truncate text-sm font-semibold">{name}</h2>
              </Tooltip>
              <p className="text-muted-foreground text-xs">
                {t("accounts.count", { count: accounts.length })}
              </p>
            </div>
            <ChevronRight className="text-muted-foreground size-4 shrink-0" aria-hidden="true" />
          </Link>
          <AddAccountDialog provider={provider} onCreate={onCreate} onBatchCreate={onBatchCreate} />
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
                    <Tooltip content={accountDisplayName(account)}>
                      <p className="truncate text-sm font-medium">{accountDisplayName(account)}</p>
                    </Tooltip>
                    <StepFunBadge account={account} />
                    <p className="text-muted-foreground text-xs">
                      {t(
                        account.authMethod === "subscription_login"
                          ? "subscription.type"
                          : "accounts.apiKey",
                      )}{" "}
                      · {account.enabled ? t("accounts.enabled") : t("accounts.disabled")}
                    </p>
                  </div>
                  <AccountItemActions
                    account={account}
                    busy={busy}
                    onToggle={onToggle}
                    onDelete={onDelete}
                    onEdit={onEdit}
                  />
                </li>
              ))}
            </ul>
          </ScrollRegion>
        )}
      </CardContent>
    </Card>
  )
}
