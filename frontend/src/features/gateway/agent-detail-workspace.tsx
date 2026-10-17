import { useCallback, useEffect, useRef, useState } from "react"
import type { AgentClient } from "../../services/agents"
import { AgentClientSettings } from "./agent-client-settings"
import { AgentInstallationCard } from "./agent-installation-card"
import { AgentConfigEditor } from "./agent-config-editor"
import { AgentConnectedModels } from "./agent-connected-models"
import { AgentConfigSuccessToast } from "./agent-config-success-toast"
import type { AgentClientId } from "./agent-clients"
import type { AgentInstallations } from "./use-agent-installations"
import "./agent-detail-workspace.css"

export function AgentDetailWorkspace({
  client,
  item,
  models,
  installations,
}: {
  client: AgentClientId
  item: AgentClient
  models: string[]
  installations: AgentInstallations
}) {
  const [editor, setEditor] = useState<{ item: AgentClient; reread: boolean } | null>(null)
  const [saved, setSaved] = useState(false)
  const root = useRef<HTMLDivElement>(null)
  const restoreFocus = useRef(false)
  useEffect(() => {
    if (!editor && restoreFocus.current) {
      restoreFocus.current = false
      root.current
        ?.querySelector<HTMLElement>("[data-agent-configure]")
        ?.focus({ preventScroll: true })
    }
  }, [editor])
  function configure(next: AgentClient, reread = false) {
    setSaved(false)
    setEditor({ item: next, reread })
  }
  const cancel = useCallback(() => {
    restoreFocus.current = true
    setEditor(null)
  }, [])
  const onSaved = useCallback(() => {
    restoreFocus.current = true
    setEditor(null)
    setSaved(true)
  }, [])
  return (
    <div ref={root} className="agent-detail-workspace">
      <div className="agent-detail-columns grid min-w-0 items-start gap-5">
        <div className="min-w-0 space-y-5">
          <AgentInstallationCard client={client} state={installations} />
          <AgentClientSettings
            item={item}
            models={models}
            onConfigure={configure}
            editing={!!editor}
          />
        </div>
        <div className="agent-detail-models relative min-w-0">
          {editor ? (
            <AgentConfigEditor
              key={`${editor.item.client}:${editor.item.revision}:${editor.reread}`}
              item={editor.item}
              reread={editor.reread}
              autoPreview
              onClose={cancel}
              onSaved={onSaved}
            />
          ) : (
            <AgentConnectedModels item={item} />
          )}
          {saved ? <AgentConfigSuccessToast /> : null}
        </div>
      </div>
    </div>
  )
}
