import { Link } from "@tanstack/react-router"
import { AlertCircle, ArrowUpRight, Check, CircleDashed, LoaderCircle } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardDescription, CardHeader } from "../../components/ui/card"
import type { TranslationKey } from "../../i18n/translation"
import type { PreparationState } from "./gateway-preparation"
import type { GatewayPreparation } from "./use-gateway-preparation"

const stateKeys: Record<PreparationState, TranslationKey> = {
  loading: "gateway.loading",
  error: "gateway.check.error",
  missing: "gateway.check.missing",
  configured: "gateway.check.configured",
}
const stateIcons = {
  loading: LoaderCircle,
  error: AlertCircle,
  missing: CircleDashed,
  configured: Check,
}

export function GatewayReadinessCard({ data }: { data: GatewayPreparation }) {
  const { t } = useTranslation()
  const checks: {
    title: TranslationKey
    hint: TranslationKey
    state: PreparationState
    count?: number | null
    to?: "/providers" | "/accounts"
    action?: TranslationKey
  }[] = [
    { title: "gateway.check.runtime", hint: "gateway.check.runtimeHint", state: data.runtimeState },
    {
      title: "gateway.check.providers",
      hint: "gateway.check.providersHint",
      state: data.providerState,
      count: data.providerCount,
      to: "/providers",
      action: "gateway.check.manageProviders",
    },
    {
      title: "gateway.check.accounts",
      hint: "gateway.check.accountsHint",
      state: data.accountState,
      count: data.accountCount,
      to: "/accounts",
      action: "gateway.check.manageAccounts",
    },
  ]

  return (
    <Card className="dark:bg-card/75 min-w-0 gap-4">
      <CardHeader>
        <h2 className="text-base font-semibold">{t("gateway.check.title")}</h2>
        <CardDescription>{t("gateway.check.description")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <ul className="grid gap-3 md:grid-cols-3">
          {checks.map(({ title, hint, state, count, to, action }) => {
            const Icon = stateIcons[state]
            return (
              <li
                key={title}
                className="bg-muted/15 flex min-w-0 flex-col gap-3 rounded-lg border p-4"
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <h3 className="text-sm font-medium">{t(title)}</h3>
                  <span
                    className={`flex items-center gap-1.5 text-xs ${state === "error" ? "text-destructive" : "text-muted-foreground"}`}
                  >
                    <Icon className="size-3.5" aria-hidden="true" />
                    {t(
                      title === "gateway.check.runtime" && state === "configured"
                        ? "gateway.check.read"
                        : stateKeys[state],
                    )}
                    {state === "configured" && count != null
                      ? ` · ${t("gateway.check.count", { count })}`
                      : null}
                  </span>
                </div>
                <p className="text-muted-foreground flex-1 text-xs leading-5">{t(hint)}</p>
                {to && action ? (
                  <Link
                    to={to}
                    className="hover:text-primary focus-visible:ring-ring/50 inline-flex w-fit items-center gap-1 rounded-sm text-xs font-medium underline-offset-4 hover:underline focus-visible:ring-4"
                  >
                    {t(action)}
                    <ArrowUpRight className="size-3.5" aria-hidden="true" />
                  </Link>
                ) : null}
              </li>
            )
          })}
        </ul>
        <p className="text-muted-foreground text-xs leading-5">{t("gateway.check.notice")}</p>
      </CardContent>
    </Card>
  )
}
