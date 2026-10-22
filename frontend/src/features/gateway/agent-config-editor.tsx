import { useEffect, useRef } from "react"
import { RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { ScrollRegion } from "../../components/common/scroll-region"
import { Button } from "../../components/ui/button"
import { Card } from "../../components/ui/card"
import type { AgentClient } from "../../services/agents"
import { useAgentConfigForm } from "./use-agent-config-form"
import { AgentCatalogPreview } from "./agent-catalog-preview"
import { AgentConfigSuccessToast } from "./agent-config-success-toast"

export function AgentConfigEditor({
  item,
  onClose,
  onSaved,
  reread = false,
  autoPreview = false,
}: {
  item: AgentClient
  onClose: () => void
  onSaved?: () => void
  reread?: boolean
  autoPreview?: boolean
}) {
  const { t } = useTranslation()
  const {
    preview,
    busy,
    error,
    success,
    current,
    acceptCurrent,
    reloaded,
    run,
    reload,
    selectedIds,
    defaultModel,
    selectModels,
    selectDefault,
    efforts,
    selectEffort,
  } = useAgentConfigForm(item, reread)
  const started = useRef(false)
  const title = useRef<HTMLHeadingElement>(null)
  useEffect(() => {
    if (autoPreview && !started.current) {
      started.current = true
      title.current?.focus()
      title.current?.parentElement?.scrollIntoView?.({ block: "nearest" })
      void run(false)
    }
  }, [autoPreview, run])
  useEffect(() => {
    if (success) onSaved?.()
  }, [success, onSaved])
  return (
    <Card
      role="region"
      aria-labelledby="agent-model-editor-title"
      className="relative max-h-[calc(100dvh-10rem)] min-w-0 gap-0 py-0"
    >
      <div className="flex flex-wrap items-center justify-between gap-3 border-b p-5">
        <h3
          ref={title}
          id="agent-model-editor-title"
          tabIndex={-1}
          className="text-base font-semibold outline-none"
        >
          {t("agents.configureModels")}
        </h3>
        <div className="flex flex-wrap items-center justify-end gap-2">
          {!success ? (
            <Button variant="outline" disabled={busy} onClick={() => void reload()}>
              <RefreshCw aria-hidden="true" />
              {t("agents.reread")}
            </Button>
          ) : null}
          <Button variant="outline" disabled={busy} onClick={onClose}>
            {t("agents.cancelConfiguration")}
          </Button>
          {!success ? (
            <Button
              disabled={
                busy ||
                (!!preview && selectedIds.length === 0) ||
                (current.errorCode === "config_changed" && !reloaded)
              }
              onClick={() => void run(!!preview)}
            >
              {t(
                busy
                  ? preview
                    ? "agents.applying"
                    : "agents.loading"
                  : preview
                    ? "agents.apply"
                    : "agents.preview",
              )}
            </Button>
          ) : null}
        </div>
      </div>
      <ScrollRegion
        as="section"
        ariaLabel={t("agents.configure")}
        className="min-h-0 flex-1"
        viewportClassName="p-5"
      >
        <div className="space-y-4">
          {item.client === "claude" ? (
            <p className="text-muted-foreground text-xs">{t("agents.cliModelHint")}</p>
          ) : null}
          {busy && !preview ? (
            <p className="text-muted-foreground text-sm">{t("agents.loading")}</p>
          ) : null}
          {preview ? (
            <AgentCatalogPreview
              plan={preview}
              expanded
              selection={{
                ids: selectedIds,
                defaultModel,
                disabled: busy || success,
                onChange: selectModels,
                onDefaultChange: selectDefault,
                efforts,
                // 档位由网关在 Messages 请求上覆盖，只对使用该协议的客户端开放。
                onEffortChange:
                  item.client === "claude" || item.client === "claude-desktop"
                    ? selectEffort
                    : undefined,
              }}
            />
          ) : null}
          {item.client === "claude" || item.client === "claude-desktop" ? (
            <p className="text-muted-foreground text-xs">{t("agents.effortHint")}</p>
          ) : null}
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
          {error ? (
            <p role="alert" className="text-destructive text-sm">
              {t(`agents.error.${error}`)}
            </p>
          ) : null}
        </div>
      </ScrollRegion>
      {success ? <AgentConfigSuccessToast /> : null}
    </Card>
  )
}
