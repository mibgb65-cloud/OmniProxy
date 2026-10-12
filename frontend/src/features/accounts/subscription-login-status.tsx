import { ExternalLink } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { useSubscriptionLogin } from "./use-subscription-login"

export function SubscriptionLoginStatus({
  login,
}: {
  login: ReturnType<typeof useSubscriptionLogin>
}) {
  const { t } = useTranslation()
  const waiting = login.view?.state === "waiting"
  const complete = login.view?.state === "completed"
  return (
    <>
      {waiting ? (
        <div role="status" className="bg-muted space-y-3 rounded-lg p-4 text-sm">
          <p>{t("subscription.waiting")}</p>
          {login.view?.userCode ? (
            <div>
              <p className="text-muted-foreground mb-1 text-xs">{t("subscription.code")}</p>
              <code className="font-mono text-lg tracking-widest select-text">
                {login.view.userCode}
              </code>
            </div>
          ) : null}
          <p className="text-muted-foreground text-xs">{t("subscription.waitHint")}</p>
          <p className="text-muted-foreground text-xs">{t("subscription.backgroundHint")}</p>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => void login.open()}
            className="cursor-pointer"
          >
            <ExternalLink className="size-3.5" />
            {t("subscription.open")}
          </Button>
        </div>
      ) : null}
      {complete ? (
        <div role="status" className="bg-muted rounded-lg p-4 text-sm">
          <p>{t("subscription.completed")}</p>
          <p className="mt-1 break-all">{login.view?.account?.authEmail}</p>
        </div>
      ) : null}
      {login.view?.state === "cancelled" ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("subscription.cancelled")}
        </p>
      ) : null}
      {login.error ? (
        <p role="alert" className="text-destructive text-sm">
          {t(`subscription.error.${login.error}`, {
            defaultValue: t("subscription.error.operation_failed"),
          })}
        </p>
      ) : null}
      {login.view?.cleanupWarning ? (
        <p role="alert" className="text-destructive text-sm">
          {t("subscription.cleanupWarning")}
        </p>
      ) : null}
    </>
  )
}
