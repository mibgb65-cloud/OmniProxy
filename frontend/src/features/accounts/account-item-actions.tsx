import { Tooltip } from "../../components/ui/tooltip"
import { SquarePen, Trash2 } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { AccountItem } from "../../services/omniproxy"
import { AccountPowerButton } from "./account-power-button"
import { accountDisplayName } from "./account-display"

export type AccountItemActionsProps = {
  account: AccountItem
  busy: boolean
  onToggle: (id: string, enabled: boolean) => void
  onDelete: (account: AccountItem) => void
  onEdit: (account: AccountItem) => void
}

export function AccountItemActions({
  account,
  busy,
  onToggle,
  onDelete,
  onEdit,
}: AccountItemActionsProps) {
  const { t } = useTranslation()
  const name = accountDisplayName(account)
  return (
    <div className="flex shrink-0 items-center gap-2">
      <AccountPowerButton
        name={name}
        enabled={account.enabled}
        disabled={busy}
        onChange={(enabled) => onToggle(account.id, enabled)}
      />
      <Tooltip content={t("accounts.edit.action")}>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          disabled={busy}
          onClick={() => onEdit(account)}
          aria-label={t("accounts.edit.label", { name })}
          className="text-muted-foreground hover:text-foreground cursor-pointer"
        >
          <SquarePen className="size-4" aria-hidden="true" />
        </Button>
      </Tooltip>
      <Tooltip content={t("accounts.delete")}>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          disabled={busy}
          onClick={() => onDelete(account)}
          aria-label={t("accounts.deleteLabel", { name })}
          className="text-muted-foreground hover:bg-destructive/10 hover:text-destructive cursor-pointer"
        >
          <Trash2 className="size-4" aria-hidden="true" />
        </Button>
      </Tooltip>
    </div>
  )
}
