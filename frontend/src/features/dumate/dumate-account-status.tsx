import { ShieldAlert, ShieldCheck } from "lucide-react"
import { useTranslation } from "react-i18next"
import type { AccountItem } from "../../services/omniproxy"

/** Cookie 账号不展示密钥预览，只说明登录态是否可用。 */
export function DuMateAccountStatus({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const active = account.authState === "authenticated"
  return (
    <div className="flex min-w-0 flex-col items-start gap-1">
      <span className="bg-muted/60 inline-flex max-w-full items-center gap-1 rounded-md px-2 py-0.5 text-xs">
        {active ? (
          <ShieldCheck className="size-3 shrink-0" aria-hidden="true" />
        ) : (
          <ShieldAlert className="text-destructive size-3 shrink-0" aria-hidden="true" />
        )}
        <span className={active ? "truncate" : "text-destructive truncate"}>
          {t(active ? "dumate.account.imported" : "dumate.account.needsReimport")}
        </span>
      </span>
      <span className="text-muted-foreground text-xs">{t("dumate.account.cookieType")}</span>
    </div>
  )
}
