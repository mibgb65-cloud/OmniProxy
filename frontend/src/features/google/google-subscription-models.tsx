import { useId } from "react"
import { ChevronDown, RefreshCw, X } from "lucide-react"
import { Popover } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { ScrollRegion } from "../../components/common/scroll-region"
import type { AccountItem } from "../../services/omniproxy"
import { useGoogleSubscriptionModels } from "./use-google-subscription-quota"
import { GoogleModelSync } from "./google-model-sync"

export function GoogleSubscriptionModels({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const titleId = useId()
  const query = useGoogleSubscriptionModels(account)
  if (account.authState !== "authenticated") return null
  const models = query.data ?? []
  const supported = models.filter((model) => model.gatewaySupported)
  return (
    <div className="min-w-0 space-y-2 border-b pb-3">
      <div className="flex items-center justify-between gap-2">
        <p className="text-xs font-medium">{t("google.models.title")}</p>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          disabled={query.isFetching}
          aria-label={t("google.models.refresh")}
          className="cursor-pointer"
          onClick={() => void query.refetch({ cancelRefetch: false })}
        >
          <RefreshCw
            className={
              query.isFetching ? "size-3 animate-spin motion-reduce:animate-none" : "size-3"
            }
            aria-hidden="true"
          />
        </Button>
      </div>
      {query.isPending ? (
        <p role="status" className="text-muted-foreground text-xs">
          {t("google.models.loading")}
        </p>
      ) : null}
      {models.length ? (
        <Popover.Root modal={false}>
          <Popover.Trigger asChild>
            <button
              type="button"
              className="text-muted-foreground hover:text-foreground focus-visible:ring-ring flex w-full cursor-pointer items-center justify-between gap-2 rounded-sm text-left text-xs outline-none focus-visible:ring-2"
              aria-label={t("google.models.show", { count: models.length })}
            >
              <span className="min-w-0 space-y-1">
                <span className="block truncate">
                  {supported
                    .slice(0, 2)
                    .map((model) => model.name)
                    .join(" · ") || t("google.models.noAdapter")}
                </span>
                <span className="block text-[10px]">
                  {t("google.models.count", { supported: supported.length, total: models.length })}
                </span>
              </span>
              <ChevronDown className="size-3.5 shrink-0" aria-hidden="true" />
            </button>
          </Popover.Trigger>
          <Popover.Portal>
            <Popover.Content
              side="bottom"
              align="start"
              sideOffset={8}
              collisionPadding={12}
              sticky="always"
              aria-labelledby={titleId}
              aria-describedby={undefined}
              className="bg-popover text-popover-foreground z-50 flex max-h-[min(380px,var(--radix-popover-content-available-height))] w-[min(380px,calc(100vw-1.5rem))] flex-col gap-3 overflow-hidden rounded-xl border p-3 shadow-xl outline-none"
            >
              <div className="flex shrink-0 items-center justify-between gap-2">
                <h4 id={titleId} className="text-xs font-medium">
                  {t("google.models.title")}
                </h4>
                <Popover.Close asChild>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    aria-label={t("google.models.close")}
                    className="cursor-pointer"
                  >
                    <X className="size-3.5" aria-hidden="true" />
                  </Button>
                </Popover.Close>
              </div>
              <ScrollRegion
                as="section"
                ariaLabel={t("google.models.title")}
                className="min-h-0 flex-1"
              >
                <div className="space-y-4">
                  {[true, false].map((connected) => {
                    const items = models.filter((model) => model.gatewaySupported === connected)
                    if (!items.length) return null
                    return (
                      <section key={String(connected)} className="space-y-2">
                        <h5 className="text-muted-foreground text-[11px] font-medium">
                          {t(connected ? "google.models.connected" : "google.models.pending")}
                        </h5>
                        <ul className="space-y-2">
                          {items.map((model) => (
                            <li
                              key={model.id}
                              className="bg-muted/40 min-w-0 rounded-md p-2 select-text"
                            >
                              <p className="text-xs font-medium wrap-anywhere">{model.name}</p>
                              <code className="text-muted-foreground block text-[10px] wrap-anywhere">
                                {model.id}
                              </code>
                            </li>
                          ))}
                        </ul>
                      </section>
                    )
                  })}
                </div>
              </ScrollRegion>
              <p className="text-muted-foreground shrink-0 text-[10px] leading-5">
                {t("google.models.hint")}
              </p>
            </Popover.Content>
          </Popover.Portal>
        </Popover.Root>
      ) : !query.isPending && !query.isError ? (
        <p className="text-muted-foreground text-xs">{t("google.models.empty")}</p>
      ) : null}
      {query.isError ? (
        <p role="alert" className="text-destructive text-xs">
          {t("google.models.failed")}
        </p>
      ) : null}
      <GoogleModelSync
        key={account.updatedAt}
        accountId={account.id}
        disabled={!supported.length || !account.enabled}
      />
    </div>
  )
}
