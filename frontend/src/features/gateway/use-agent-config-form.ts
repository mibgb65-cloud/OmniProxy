import { useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import {
  agentError,
  applyAgentConfig,
  previewAgentPlan,
  rereadAgentConfig,
  type AgentPlan,
  type AgentClient,
} from "../../services/agents"

export function useAgentConfigForm(
  item: AgentClient,
  reread: boolean,
): {
  preview: AgentPlan | null
  busy: boolean
  error: ReturnType<typeof agentError> | ""
  success: boolean
  current: AgentClient
  acceptCurrent: boolean
  reloaded: boolean
  run: (apply: boolean) => Promise<void>
  reload: () => Promise<void>
  selectedIds: string[]
  defaultModel: string
  selectModels: (ids: string[]) => void
  selectDefault: (id: string) => void
} {
  const client = useQueryClient()
  const [preview, setPreview] = useState<AgentPlan | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<ReturnType<typeof agentError> | "">("")
  const [success, setSuccess] = useState(false)
  const [current, setCurrent] = useState(item)
  const [acceptCurrent, setAcceptCurrent] = useState(reread && item.errorCode === "config_changed")
  const [reloaded, setReloaded] = useState(reread)
  const [modelIds, setModelIds] = useState<string[] | null>(null)
  const [model, setModel] = useState("")
  const selectedModels = (preview?.models ?? []).filter(
    (entry) => modelIds === null || modelIds.includes(entry.id),
  )
  const defaultModel =
    selectedModels.find((entry) => entry.id === (model || preview?.defaultModel))?.id ??
    selectedModels[0]?.id ??
    ""
  function selectModels(ids: string[]) {
    setModelIds(ids.length === preview?.models?.length ? null : ids)
  }
  function resetSelection() {
    setModelIds(null)
    setModel("")
  }
  async function reload() {
    setBusy(true)
    setError("")
    setPreview(null)
    resetSelection()
    setSuccess(false)
    try {
      const next = await rereadAgentConfig(current.client)
      setCurrent(next)
      setAcceptCurrent(next.errorCode === "config_changed")
      setReloaded(true)
      await client.invalidateQueries({ queryKey: ["agent-clients"] })
    } catch (error) {
      setError(agentError(error))
    } finally {
      setBusy(false)
    }
  }
  async function run(apply: boolean) {
    setBusy(true)
    setError("")
    const input = {
      client: current.client,
      revision: current.revision,
      ...(acceptCurrent ? { acceptCurrentConfig: true } : {}),
      ...(apply && preview ? { catalogRevision: preview.catalogRevision } : {}),
      ...(apply && modelIds !== null ? { modelIds } : {}),
      ...(apply && defaultModel && (model || modelIds !== null) ? { model: defaultModel } : {}),
    }
    try {
      if (apply) {
        await applyAgentConfig(input)
        setSuccess(true)
        await client.invalidateQueries({ queryKey: ["agent-clients"] })
        await client.invalidateQueries({ queryKey: ["agent-configured-models"] })
      } else setPreview(await previewAgentPlan(input))
    } catch (error) {
      const code = agentError(error)
      setError(code)
      if (
        code === "config_changed" ||
        code === "catalog_changed" ||
        code === "no_models" ||
        code === "models_unavailable"
      ) {
        setPreview(null)
        resetSelection()
        if (code === "config_changed") {
          setAcceptCurrent(false)
          setReloaded(false)
        }
      }
    } finally {
      setBusy(false)
    }
  }
  return {
    preview,
    busy,
    error,
    success,
    current,
    acceptCurrent,
    reloaded,
    run,
    reload,
    selectedIds: selectedModels.map((entry) => entry.id),
    defaultModel,
    selectModels,
    selectDefault: setModel,
  }
}
