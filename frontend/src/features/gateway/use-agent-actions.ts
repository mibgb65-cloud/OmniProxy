import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  agentError,
  copyAgentKey,
  restoreAgentConfig,
  setAgentEnabled,
  type AgentClient,
} from "../../services/agents"

type AgentAction = "copy" | "toggle" | "restore"

export function useAgentActions(item: AgentClient): {
  busy: boolean
  error: ReturnType<typeof agentError> | null
  notice: "agents.keyCopied" | "agents.restored" | null
  run: (action: AgentAction) => void
} {
  const client = useQueryClient()
  const operation = useMutation({
    mutationFn: async (action: AgentAction) => {
      if (action === "copy") await copyAgentKey(item.client)
      else if (action === "restore") await restoreAgentConfig(item.client)
      else await setAgentEnabled(item.client, !item.configured)
    },
    onSuccess: async (_, action) => {
      if (action !== "copy") {
        await client.invalidateQueries({ queryKey: ["agent-clients"] })
        await client.invalidateQueries({ queryKey: ["agent-configured-models"] })
      }
    },
  })
  const notice =
    operation.isSuccess && operation.variables === "copy"
      ? "agents.keyCopied"
      : operation.isSuccess && operation.variables === "restore"
        ? "agents.restored"
        : null
  return {
    busy: operation.isPending,
    error: operation.isError ? agentError(operation.error) : null,
    notice,
    run: (action: AgentAction) => {
      if (!operation.isPending) operation.mutate(action)
    },
  }
}
