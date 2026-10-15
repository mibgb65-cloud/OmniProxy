import { useState } from "react"
import { useTranslation } from "react-i18next"
import { ExternalLink } from "lucide-react"
import { Button } from "../../components/ui/button"
import { openMiMoConsole } from "../../services/mimo"
import type { AccountItem } from "../../services/omniproxy"

export function MiMoPlanResource({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const [failed, setFailed] = useState(false)
  return (
    <div className="bg-muted/20 space-y-3 rounded-xl border p-3">
      <p className="text-sm font-medium">{t("mimo.quotaTitle")}</p>
      <p className="text-muted-foreground text-xs leading-5">{t("mimo.quotaUnknown")}</p>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => {
          setFailed(false)
          void openMiMoConsole(account.id).catch(() => setFailed(true))
        }}
      >
        <ExternalLink className="size-3.5" aria-hidden="true" />
        {t("mimo.openConsole")}
      </Button>
      {failed ? (
        <p role="alert" className="text-destructive text-xs">
          {t("mimo.openFailed")}
        </p>
      ) : null}
    </div>
  )
}
