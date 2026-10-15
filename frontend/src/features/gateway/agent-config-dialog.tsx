import { Dialog } from "radix-ui"
import { RefreshCw, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { ScrollRegion } from "../../components/common/scroll-region"
import { Button } from "../../components/ui/button"
import { agentTitle, type AgentClient } from "../../services/agents"
import { useAgentConfigForm } from "./use-agent-config-form"
import { AgentCatalogPreview } from "./agent-catalog-preview"
import { AgentConfigSuccessToast } from "./agent-config-success-toast"

export function AgentConfigDialog({
  item,
  onClose,
  reread = false,
}: {
  item: AgentClient | null
  onClose: () => void
  reread?: boolean
}) {
  return (
    <Dialog.Root
      open={!!item}
      onOpenChange={(open) => {
        if (!open) onClose()
      }}
    >
      {item ? (
        <ConfigForm
          key={`${item.client}:${item.revision}:${reread}`}
          item={item}
          onClose={onClose}
          reread={reread}
        />
      ) : null}
    </Dialog.Root>
  )
}
function ConfigForm({
  item,
  onClose,
  reread,
}: {
  item: AgentClient
  onClose: () => void
  reread: boolean
}) {
  const { t } = useTranslation()
  const { preview, busy, error, success, current, acceptCurrent, reloaded, run, reload } =
    useAgentConfigForm(item, reread)
  return (
    <Dialog.Portal>
      <Dialog.Overlay className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm" />
      <Dialog.Content
        aria-describedby="agent-config-description"
        onEscapeKeyDown={(e) => {
          if (busy) e.preventDefault()
        }}
        onInteractOutside={(e) => {
          if (busy) e.preventDefault()
        }}
        className="bg-popover text-popover-foreground fixed top-1/2 left-1/2 z-50 flex max-h-[calc(100dvh-3rem)] w-[min(800px,calc(100vw-3rem))] -translate-x-1/2 -translate-y-1/2 flex-col gap-5 rounded-xl border p-6 shadow-2xl outline-none"
      >
        <div className="flex shrink-0 items-center justify-between gap-3 border-b pb-3">
          <Dialog.Title className="text-base font-semibold">
            {t("agents.configure")} · {t(agentTitle(item.client))}
          </Dialog.Title>
          <Button
            variant="ghost"
            size="icon-sm"
            disabled={busy}
            aria-label={t("agents.close")}
            onClick={onClose}
          >
            <X />
          </Button>
        </div>
        <ScrollRegion
          as="section"
          ariaLabel={t("agents.configure")}
          className="min-h-0 flex-1"
          viewportClassName="pr-1"
        >
          <div className="space-y-5">
            {item.client === "claude" ? (
              <p className="text-muted-foreground text-xs">{t("agents.cliModelHint")}</p>
            ) : null}
            {preview ? <AgentCatalogPreview plan={preview} /> : null}
            {reloaded ? (
              <p
                role="status"
                className="text-muted-foreground rounded-lg border p-3 text-sm leading-6"
              >
                {t(acceptCurrent ? "agents.rereadChangesNotice" : "agents.rereadSuccess")}
              </p>
            ) : null}
            {!reloaded && current.errorCode === "config_changed" ? (
              <p role="alert" className="text-destructive text-sm">
                {t("agents.error.config_changed")}
              </p>
            ) : null}
            {item.client === "claude-desktop" ? (
              <p className="text-muted-foreground rounded-lg border p-3 text-sm leading-6">
                {t("agents.desktopHint")}
              </p>
            ) : null}
            <Dialog.Description
              id="agent-config-description"
              className="text-muted-foreground text-xs"
            >
              {t("agents.restart")}
            </Dialog.Description>
            {error ? (
              <p role="alert" className="text-destructive text-sm">
                {t(`agents.error.${error}`)}
              </p>
            ) : null}
          </div>
        </ScrollRegion>
        {success ? <AgentConfigSuccessToast /> : null}
        <div className="flex shrink-0 justify-end gap-3 border-t pt-4">
          {!success ? (
            <Button variant="outline" disabled={busy} onClick={() => void reload()}>
              <RefreshCw aria-hidden="true" />
              {t("agents.reread")}
            </Button>
          ) : null}
          <Button variant="outline" disabled={busy} onClick={onClose}>
            {t("agents.close")}
          </Button>
          {!success ? (
            <Button
              disabled={busy || (current.errorCode === "config_changed" && !reloaded)}
              onClick={() => void run(!!preview)}
            >
              {t(busy ? "agents.applying" : preview ? "agents.apply" : "agents.preview")}
            </Button>
          ) : null}
        </div>
      </Dialog.Content>
    </Dialog.Portal>
  )
}
