import { useMutation, useQuery } from "@tanstack/react-query"
import { Clipboard } from "@wailsio/runtime"
import { Dialog } from "radix-ui"
import { Copy, ScrollText, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import { formatTokens, readRequest } from "../../services/activity"
import { stateLabel, requestTime } from "./activity-format"
import { RequestAttempts } from "./request-attempts"

export function RequestDialog({
  id,
  onClose,
  onOpenLogs,
}: {
  id: string | null
  onClose: () => void
  onOpenLogs?: (id: string) => void
}) {
  const { t, i18n } = useTranslation()
  const data = useQuery({
    queryKey: ["request", id],
    queryFn: () => readRequest(id || ""),
    enabled: !!id,
    refetchInterval: id ? 5000 : false,
    retry: false,
  })
  const copy = useMutation({ mutationFn: () => Clipboard.SetText(id || "") })
  const item = data.data
  return (
    <Dialog.Root
      open={!!id}
      onOpenChange={(open) => {
        if (!open) onClose()
      }}
    >
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm" />
        <Dialog.Content
          aria-describedby={undefined}
          className="bg-popover text-popover-foreground fixed top-1/2 left-1/2 z-50 flex max-h-[calc(100dvh-3rem)] w-[min(800px,calc(100vw-3rem))] -translate-x-1/2 -translate-y-1/2 flex-col gap-5 rounded-xl border p-6 shadow-2xl outline-none"
        >
          <div className="flex shrink-0 items-center justify-between gap-3 border-b pb-3">
            <Dialog.Title className="text-base font-semibold">{t("activity.details")}</Dialog.Title>
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label={t("activity.close")}
              onClick={onClose}
            >
              <X aria-hidden="true" />
            </Button>
          </div>
          <div className="min-h-0 flex-1 space-y-4 overflow-auto pr-1">
            <div className="flex items-center justify-between gap-2 rounded-lg border p-3">
              <code className="min-w-0 text-xs wrap-anywhere">{id}</code>
              <Tooltip content={t(copy.isSuccess ? "activity.copied" : "activity.copyId")}>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={t("activity.copyId")}
                  disabled={copy.isPending}
                  onClick={() => copy.mutate()}
                >
                  <Copy aria-hidden="true" />
                </Button>
              </Tooltip>
            </div>
            {data.isPending ? (
              <p role="status" className="text-muted-foreground text-sm">
                {t("activity.loading")}
              </p>
            ) : data.isError ? (
              <p role="alert" className="text-destructive text-sm">
                {t("activity.detailGone")}
              </p>
            ) : item ? (
              <dl className="grid gap-4 sm:grid-cols-2">
                {(
                  [
                    ["activity.state", t(stateLabel(item.state))],
                    ["activity.time", requestTime(item.startedAt)],
                    ["activity.protocol", `${item.protocol} / ${item.transport}`],
                    ["activity.model", item.model || item.inputModel || "—"],
                    ["activity.provider", item.providerName || "—"],
                    ...(item.providerRegion
                      ? ([
                          [
                            "stepfun.region",
                            t(item.providerRegion === "cn" ? "stepfun.cn" : "stepfun.global"),
                          ],
                          [
                            "stepfun.billing",
                            t(item.billingMode === "step_plan" ? "stepfun.plan" : "stepfun.payg"),
                          ],
                        ] as const)
                      : []),
                    ["activity.account", item.email || item.accountName || "—"],
                    ["activity.workspace", item.workspaceId || "—"],
                    ["activity.httpStatus", item.statusCode || "—"],
                    [
                      "activity.input",
                      item.inputKnown
                        ? formatTokens(item.inputTokens, i18n.resolvedLanguage)
                        : t("activity.unknown"),
                    ],
                    [
                      "activity.output",
                      item.outputKnown
                        ? formatTokens(item.outputTokens, i18n.resolvedLanguage)
                        : t("activity.unknown"),
                    ],
                    [
                      "activity.usageState",
                      t(
                        item.usageComplete
                          ? "activity.complete"
                          : item.inputKnown || item.outputKnown
                            ? "activity.partial"
                            : "activity.unknown",
                      ),
                    ],
                    [
                      "activity.firstToken",
                      item.firstTokenMs == null ? "—" : `${item.firstTokenMs} ms`,
                    ],
                    ["activity.latency", item.state === "pending" ? "—" : `${item.latencyMs} ms`],
                    ["activity.errorCode", item.errorCode || "—"],
                  ] as const
                ).map(([label, value]) => (
                  <div key={label}>
                    <dt className="text-muted-foreground text-xs">{t(label)}</dt>
                    <dd className="mt-1 text-sm wrap-anywhere">{value}</dd>
                  </div>
                ))}
              </dl>
            ) : null}
            {data.isSuccess && item ? <RequestAttempts item={item} /> : null}
            <p className="text-muted-foreground text-xs leading-5">{t("activity.logHint")}</p>
          </div>
          <div className="flex shrink-0 justify-end gap-3 border-t pt-4">
            <Button variant="outline" onClick={onClose}>
              {t("activity.close")}
            </Button>
            {onOpenLogs && id ? (
              <Button
                onClick={() => {
                  onOpenLogs(id)
                  onClose()
                }}
              >
                <ScrollText aria-hidden="true" />
                {t("activity.logs")}
              </Button>
            ) : null}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
