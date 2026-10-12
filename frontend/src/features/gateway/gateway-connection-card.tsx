import { useMutation } from "@tanstack/react-query"
import { Clipboard } from "@wailsio/runtime"
import { Check, Copy } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardDescription, CardHeader } from "../../components/ui/card"
import type { GatewayPreparation } from "./use-gateway-preparation"

export function GatewayConnectionCard({ data }: { data: GatewayPreparation }) {
  const { t } = useTranslation()
  const copy = useMutation({ mutationFn: (address: string) => Clipboard.SetText(address) })
  const copied = copy.isSuccess && copy.variables === data.address
  const copyFailed = copy.isError && copy.variables === data.address

  return (
    <Card className="dark:bg-card/75 min-w-0 gap-4">
      <CardHeader>
        <h2 className="text-base font-semibold">{t("gateway.connection.title")}</h2>
        <CardDescription>{t("gateway.connection.description")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4 text-sm">
        <div>
          <p className="text-muted-foreground mb-2 text-xs">{t("gateway.connection.address")}</p>
          <div className="bg-muted/30 flex min-w-0 items-center gap-3 rounded-lg border p-3">
            <code className="min-w-0 flex-1 text-sm wrap-anywhere">
              {data.address ?? (data.runtime.isPending ? t("gateway.loading") : "—")}
            </code>
            <Button
              variant="outline"
              size="icon-sm"
              className="size-9"
              disabled={!data.address || copy.isPending}
              aria-label={t(
                copy.isPending ? "gateway.connection.copying" : "gateway.connection.copy",
              )}
              onClick={() => data.address && copy.mutate(data.address)}
            >
              {copied ? <Check aria-hidden="true" /> : <Copy aria-hidden="true" />}
            </Button>
          </div>
        </div>
        {copied ? <p role="status">{t("gateway.connection.copied")}</p> : null}
        {copyFailed ? (
          <p role="alert" className="text-destructive">
            {t("gateway.connection.copyFailed")}
          </p>
        ) : null}
        <p className="text-muted-foreground text-xs leading-5">
          {t(
            data.gateway.state === "unknown"
              ? "gateway.status.loadFailed"
              : data.gateway.state === "loading"
                ? "gateway.loading"
                : data.gateway.isRunning
                  ? "gateway.connection.runningNotice"
                  : "gateway.connection.notice",
          )}
        </p>
      </CardContent>
    </Card>
  )
}
