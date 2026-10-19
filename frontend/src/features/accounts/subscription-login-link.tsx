import { useMutation } from "@tanstack/react-query"
import { Clipboard } from "@wailsio/runtime"
import { Check, Copy, ExternalLink } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"

export function SubscriptionLoginLink({ url, open }: { url: string; open: () => Promise<void> }) {
  const { t } = useTranslation()
  const copy = useMutation({ mutationFn: () => Clipboard.SetText(url), gcTime: 0 })
  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => void open()}
          className="min-w-0 flex-1 cursor-pointer"
        >
          <ExternalLink className="size-3.5" aria-hidden="true" />
          {t("subscription.open")}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={!url || copy.isPending}
          onClick={() => copy.mutate()}
          className="cursor-pointer"
        >
          {copy.isSuccess ? (
            <Check className="size-3.5" aria-hidden="true" />
          ) : (
            <Copy className="size-3.5" aria-hidden="true" />
          )}
          {t(copy.isSuccess ? "subscription.linkCopied" : "subscription.copyLink")}
        </Button>
      </div>
      <p className="text-muted-foreground text-xs leading-5">{t("subscription.copyLinkHint")}</p>
      {copy.isSuccess ? (
        <p role="status" className="text-muted-foreground text-xs">
          {t("subscription.linkCopied")}
        </p>
      ) : null}
      {copy.isError ? (
        <p role="alert" className="text-destructive text-xs">
          {t("subscription.copyLinkFailed")}
        </p>
      ) : null}
    </div>
  )
}
