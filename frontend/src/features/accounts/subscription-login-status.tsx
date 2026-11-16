import { AlertCircle, Check, LoaderCircle } from "lucide-react"
import { useTranslation } from "react-i18next"
import type { useSubscriptionLogin } from "./use-subscription-login"
import { subscriptionInProgress } from "../../services/subscription"
import { workbuddyError } from "../../services/workbuddy"
import { SubscriptionLoginLink } from "./subscription-login-link"

export function SubscriptionLoginStatus({
  login,
  errorNamespace = "subscription",
}: {
  login: ReturnType<typeof useSubscriptionLogin>
  /** WorkBuddy 的授权错误有自己的稳定码与文案，其余提供商沿用订阅错误字典。 */
  errorNamespace?: "subscription" | "workbuddy"
}) {
  const { t } = useTranslation()
  const waiting = login.view?.state === "waiting"
  const inProgress = subscriptionInProgress(login.view?.state)
  const phase = login.view?.state
  const title =
    phase === "verifying"
      ? "subscription.verifying"
      : phase === "saving"
        ? "subscription.saving"
        : phase === "exchanging"
          ? "subscription.exchanging"
          : "subscription.waiting"
  const complete = login.view?.state === "completed"
  return (
    <>
      {login.starting && !inProgress ? (
        <div
          role="status"
          className="text-muted-foreground flex items-center gap-3 rounded-xl border p-4 text-sm"
        >
          <LoaderCircle
            className="size-4 shrink-0 animate-spin motion-reduce:animate-none"
            aria-hidden="true"
          />
          {t("subscription.starting")}
        </div>
      ) : null}
      {inProgress ? (
        <div
          role="status"
          aria-atomic="true"
          className="bg-muted/25 space-y-4 rounded-xl border p-4"
        >
          <div className="flex items-start gap-3">
            <span className="bg-background flex size-9 shrink-0 items-center justify-center rounded-lg border">
              <LoaderCircle
                className="size-4 animate-spin motion-reduce:animate-none"
                aria-hidden="true"
              />
            </span>
            <div className="space-y-1">
              <p className="text-sm font-medium">{t(title)}</p>
              <p className="text-muted-foreground text-xs leading-5">
                {t(waiting ? "subscription.waitHint" : "subscription.processingHint")}
              </p>
            </div>
          </div>
          {waiting && login.view?.userCode ? (
            <div className="bg-background space-y-2 rounded-lg border border-dashed px-4 py-3 text-center">
              <p className="text-muted-foreground text-xs">{t("subscription.code")}</p>
              <code className="block font-mono text-2xl font-semibold tracking-[0.16em] break-all select-text">
                {login.view.userCode}
              </code>
            </div>
          ) : null}
          {waiting && login.view?.url ? (
            <SubscriptionLoginLink key={login.view.id} url={login.view.url} open={login.open} />
          ) : null}
          <p className="text-muted-foreground border-t pt-3 text-xs leading-5">
            {t("subscription.backgroundHint")}
          </p>
        </div>
      ) : null}
      {complete ? (
        <div
          role="status"
          aria-atomic="true"
          className="flex items-start gap-3 rounded-xl border border-emerald-600/25 bg-emerald-500/5 p-4"
        >
          <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
            <Check className="size-4" aria-hidden="true" />
          </span>
          <div className="min-w-0 space-y-1">
            <p className="text-sm font-medium">{t("subscription.completed")}</p>
            <p className="text-muted-foreground text-sm break-all">
              {login.view?.account?.authEmail}
            </p>
          </div>
        </div>
      ) : null}
      {login.view?.state === "cancelled" ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("subscription.cancelled")}
        </p>
      ) : null}
      {login.error ? (
        <div
          role="alert"
          className="border-destructive/20 bg-destructive/5 text-destructive flex items-start gap-2.5 rounded-xl border p-3 text-sm leading-6"
        >
          <AlertCircle className="mt-1 size-4 shrink-0" aria-hidden="true" />
          <p>
            {errorNamespace === "workbuddy"
              ? t(`workbuddy.error.${workbuddyError(login.error)}`, {
                  defaultValue: t("workbuddy.error.operation_failed"),
                })
              : t(`subscription.error.${login.error}`, {
                  defaultValue: t("subscription.error.operation_failed"),
                })}
          </p>
        </div>
      ) : null}
      {login.view?.cleanupWarning ? (
        <p
          role="alert"
          className="border-destructive/20 bg-destructive/5 text-destructive rounded-xl border p-3 text-sm leading-6"
        >
          {t("subscription.cleanupWarning")}
        </p>
      ) : null}
    </>
  )
}
