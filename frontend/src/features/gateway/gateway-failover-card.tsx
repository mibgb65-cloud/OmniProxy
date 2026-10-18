import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardDescription, CardHeader } from "../../components/ui/card"
import { Button } from "../../components/ui/button"
import { ProviderSwitch } from "../providers/provider-switch"
import { getFailoverSettings, getFailoverHealth, setFailoverEnabled } from "../../services/failover"
import { gatewayErrorKey } from "./gateway-errors"
import { requestTime } from "../activity/activity-format"

export function GatewayFailoverCard({ active, running }: { active: boolean; running: boolean }) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const settings = useQuery({
    queryKey: ["gateway-failover-settings"],
    queryFn: getFailoverSettings,
    enabled: active,
    retry: false,
  })
  const health = useQuery({
    queryKey: ["gateway-failover-health"],
    queryFn: getFailoverHealth,
    enabled: active,
    refetchInterval: active && running ? 5000 : false,
    retry: false,
  })
  const save = useMutation({
    mutationFn: setFailoverEnabled,
    onSuccess: (value) => client.setQueryData(["gateway-failover-settings"], value),
  })
  const items = health.data ?? []
  return (
    <Card className="min-w-0 gap-4">
      <CardHeader>
        <div className="flex items-center justify-between gap-4">
          <h2 className="text-base font-semibold">{t("failover.title")}</h2>
          <ProviderSwitch
            name={t("failover.title")}
            ariaLabel={t("failover.title")}
            checked={settings.data?.enabled ?? false}
            disabled={!settings.data || save.isPending}
            onChange={(enabled) => save.mutate(enabled)}
          />
        </div>
        <CardDescription>{t("failover.description")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-muted-foreground text-xs leading-5">
          {settings.data
            ? t("failover.boundary", { count: settings.data.maxAttempts })
            : t("gateway.loading")}
        </p>
        {save.isPending ? (
          <p role="status" className="text-muted-foreground text-xs">
            {t("failover.saving")}
          </p>
        ) : null}
        {settings.isError || save.isError || health.isError ? (
          <div className="space-y-2">
            <p role="alert" className="text-destructive text-xs">
              {t("failover.failed")}
            </p>
            <Button
              size="sm"
              variant="outline"
              onClick={() => {
                save.reset()
                void settings.refetch()
                void health.refetch()
              }}
            >
              {t("failover.retry")}
            </Button>
          </div>
        ) : null}
        {health.isSuccess ? (
          <p aria-live="polite" aria-atomic="true" className="text-muted-foreground text-xs">
            {t(items.length ? "failover.cooling" : "failover.healthy", { count: items.length })}
          </p>
        ) : null}
        {items.length ? (
          <ul className="space-y-2">
            {items.slice(0, 8).map((item) => (
              <li
                key={`${item.accountId}/${item.model}/${item.retryAt}`}
                className="space-y-1 rounded-lg border p-3 text-xs"
              >
                <p className="font-medium wrap-anywhere">
                  {item.accountName || t("failover.account")} · {item.model}
                </p>
                <p className="text-muted-foreground">{t(gatewayErrorKey(item.code))}</p>
                <p className="text-muted-foreground">
                  {item.recovering
                    ? t("failover.recovering")
                    : t("failover.until", { time: requestTime(item.retryAt) })}
                </p>
              </li>
            ))}
            {items.length > 8 ? (
              <li className="text-muted-foreground text-xs">
                {t("failover.more", { count: items.length - 8 })}
              </li>
            ) : null}
          </ul>
        ) : null}
      </CardContent>
    </Card>
  )
}
