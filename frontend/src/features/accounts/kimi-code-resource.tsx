import { useState } from "react"
import { useTranslation } from "react-i18next"
import { ExternalLink } from "lucide-react"
import { Button } from "../../components/ui/button"
import { openKimiCodeConsole } from "../../services/kimi-settings"
import type { AccountItem } from "../../services/omniproxy"

export function KimiCodeResource({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const [failed, setFailed] = useState(false)
  return (
    <div className="bg-muted/20 space-y-3 rounded-xl border p-3">
      <p className="text-sm font-medium">{t("kimi.codeQuota")}</p>
      <p className="text-muted-foreground text-xs leading-5">{t("kimi.codeUnknown")}</p>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => {
          setFailed(false)
          void openKimiCodeConsole(account.id).catch(() => setFailed(true))
        }}
      >
        <ExternalLink className="size-3.5" aria-hidden="true" />
        {t("kimi.openCodeConsole")}
      </Button>
      {failed ? (
        <p role="alert" className="text-destructive text-xs">
          {t("kimi.openCodeFailed")}
        </p>
      ) : null}
    </div>
  )
}
