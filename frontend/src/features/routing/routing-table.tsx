import { Pencil, Trash2 } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { ProviderSwitch } from "../providers/provider-switch"
import type { RouteItem } from "../../services/routing"
import { routingErrorKey } from "./routing-errors"

const columns = [
  "routing.status",
  "routing.alias",
  "routing.target",
  "routing.account",
  "routing.actions",
] as const

export function RoutingTable({
  items,
  busy,
  onEdit,
  onDelete,
  onToggle,
}: {
  items: RouteItem[]
  busy: boolean
  onEdit: (item: RouteItem) => void
  onDelete: (item: RouteItem) => void
  onToggle: (id: string, enabled: boolean) => void
}) {
  const { t } = useTranslation()
  return (
    <div className="overflow-x-auto rounded-xl border">
      <table aria-label={t("routing.table")} className="w-full min-w-[650px] text-sm">
        <thead className="bg-muted/30 text-muted-foreground">
          <tr>
            {columns.map((key) => (
              <th key={key} scope="col" className="px-4 py-3 text-left text-xs font-medium">
                {t(key)}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-border divide-y">
          {items.map((item) => (
            <tr key={item.rule.id} className="hover:bg-muted/15">
              <td className="px-4 py-4">
                <div className="flex items-center gap-2">
                  <ProviderSwitch
                    name={item.rule.alias}
                    ariaLabel={t("routing.toggle", { alias: item.rule.alias })}
                    checked={item.rule.enabled}
                    disabled={busy}
                    onChange={(enabled) => onToggle(item.rule.id, enabled)}
                  />
                  <span className="text-xs">
                    {t(item.rule.enabled ? "routing.enabled" : "routing.disabled")}
                  </span>
                </div>
              </td>
              <th
                scope="row"
                className="max-w-48 px-4 py-4 text-left font-mono font-medium wrap-anywhere"
              >
                {item.rule.alias}
              </th>
              <td className="px-4 py-4">
                <div className="space-y-1">
                  <p className="font-medium">{item.providerName || "—"}</p>
                  <code className="text-muted-foreground text-xs">{item.rule.model}</code>
                  {item.errorCode ? (
                    <p className="text-destructive text-xs">{t(routingErrorKey(item.errorCode))}</p>
                  ) : null}
                </div>
              </td>
              <td className="max-w-48 px-4 py-4">
                <p className="text-xs">
                  {t(item.rule.accountId ? "routing.fixed" : "routing.pool")}
                </p>
                {item.rule.accountId ? (
                  <p className="text-muted-foreground mt-1 text-xs wrap-anywhere">
                    {item.accountName || "—"}
                  </p>
                ) : null}
              </td>
              <td className="px-4 py-4">
                <div className="flex gap-1">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    disabled={busy}
                    aria-label={t("routing.edit", { alias: item.rule.alias })}
                    onClick={() => onEdit(item)}
                  >
                    <Pencil aria-hidden="true" />
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    disabled={busy}
                    aria-label={t("routing.delete", { alias: item.rule.alias })}
                    className="text-muted-foreground hover:text-destructive"
                    onClick={() => onDelete(item)}
                  >
                    <Trash2 aria-hidden="true" />
                  </Button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
