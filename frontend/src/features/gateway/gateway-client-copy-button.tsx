import { useMutation } from "@tanstack/react-query"
import { Clipboard } from "@wailsio/runtime"
import { Check, Copy } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"

export function GatewayClientCopyButton({ value, label }: { value: string | null; label: string }) {
  const { t } = useTranslation()
  const copy = useMutation({ mutationFn: (text: string) => Clipboard.SetText(text) })
  const copied = copy.isSuccess && copy.variables === value
  const failed = copy.isError && copy.variables === value

  return (
    <div className="shrink-0">
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={!value || copy.isPending}
        aria-label={copy.isPending ? t("gateway.connection.copying") : label}
        onClick={() => value && copy.mutate(value)}
      >
        {copied ? <Check aria-hidden="true" /> : <Copy aria-hidden="true" />}
        {copy.isPending
          ? t("gateway.connection.copying")
          : copied
            ? t("gateway.clients.copied")
            : t("gateway.clients.copy")}
      </Button>
      {copied ? (
        <span role="status" className="sr-only">
          {t("gateway.clients.copySuccess", { label })}
        </span>
      ) : null}
      {failed ? (
        <p role="alert" className="text-destructive mt-2 max-w-48 text-xs">
          {t("gateway.connection.copyFailed")}
        </p>
      ) : null}
    </div>
  )
}
