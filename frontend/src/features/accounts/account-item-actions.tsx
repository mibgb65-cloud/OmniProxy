import { Trash2 } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { AccountItem } from "../../services/omniproxy"
import { ProviderSwitch } from "../providers/provider-switch"

export type AccountItemActionsProps = {
  account: AccountItem
  busy: boolean
  onToggle: (id: string, enabled: boolean) => void
  onDelete: (account: AccountItem) => void
}

export function AccountItemActions({ account, busy, onToggle, onDelete }: AccountItemActionsProps) {
  const { t } = useTranslation()
  return (
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
        aria-label={t("accounts.deleteLabel", { name: account.name })}
        title={t("accounts.delete")}
        className="text-muted-foreground hover:bg-destructive/10 hover:text-destructive cursor-pointer"
      >
        <Trash2 className="size-4" aria-hidden="true" />
      </Button>
    </div>
  )
}
