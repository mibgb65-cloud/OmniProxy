import { ArrowRight, Search } from "lucide-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardDescription, CardHeader } from "../../components/ui/card"
import { Input } from "../../components/ui/input"
import { routingErrorKey } from "./routing-errors"
import type { RoutingData } from "./use-routing"

export function RoutingPreview({ data }: { data: RoutingData }) {
  const { t } = useTranslation()
  const [model, setModel] = useState("")
  const valid = /^[A-Za-z0-9_.:/-]{1,128}$/.test(model)
  const result = data.preview.data
  return (
    <Card className="dark:bg-card/75 min-w-0 gap-4">
      <CardHeader>
        <h2 className="text-base font-semibold">{t("routing.preview.title")}</h2>
        <CardDescription>{t("routing.preview.description")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <form
          className="flex flex-wrap items-end gap-3"
          onSubmit={(event) => {
            event.preventDefault()
            if (valid && !data.preview.isPending) data.preview.mutate(model)
          }}
        >
          <label htmlFor="route-preview-model" className="min-w-0 flex-1 space-y-2 text-sm">
            <span>{t("routing.alias")}</span>
            <Input
              id="route-preview-model"
              placeholder="coding-fast"
              maxLength={128}
              value={model}
              onChange={(event) => {
                setModel(event.target.value)
                data.preview.reset()
              }}
            />
          </label>
          <Button type="submit" variant="outline" disabled={!valid || data.preview.isPending}>
            <Search aria-hidden="true" />
            {t(data.preview.isPending ? "routing.preview.pending" : "routing.preview.action")}
          </Button>
        </form>
        {data.preview.isError ? (
          <p role="alert" className="text-destructive text-sm">
            {t("routing.actionFailed")}
          </p>
        ) : null}
        {result ? (
          <div className="bg-muted/20 space-y-3 rounded-lg border p-4" role="status">
            <p className="text-sm font-medium">
              {t(
                result.matched
                  ? "routing.preview.matched"
                  : result.preferenceApplied
                    ? "routing.defaults.preferredMatch"
                    : result.routable
                      ? "routing.preview.direct"
                      : "routing.preview.noMatch",
              )}
            </p>
            {result.preferredProviderId &&
            !result.preferenceApplied &&
            !result.matched &&
            result.routable ? (
              <p className="text-xs text-amber-700 dark:text-amber-400">
                {t("routing.defaults.fallback", { provider: result.providerName })}
              </p>
            ) : null}
            <div className="flex flex-wrap items-center gap-2 text-sm">
              <code className="wrap-anywhere">{result.inputModel}</code>
              <ArrowRight aria-hidden="true" className="text-muted-foreground size-4" />
              <span>{result.providerName || "—"}</span>
              <ArrowRight aria-hidden="true" className="text-muted-foreground size-4" />
              <code className="wrap-anywhere">{result.model}</code>
            </div>
            {result.routable ? (
              <div className="text-muted-foreground space-y-1 text-xs">
                <p>{t("routing.preview.selected", { name: result.accountName })}</p>
                <p>{t("routing.preview.candidates", { count: result.candidateCount })}</p>
              </div>
            ) : (
              <p className="text-destructive text-sm">{t(routingErrorKey(result.errorCode))}</p>
            )}
          </div>
        ) : null}
        <p className="text-muted-foreground text-xs leading-5">{t("routing.preview.notice")}</p>
      </CardContent>
    </Card>
  )
}
