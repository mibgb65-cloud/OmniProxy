import { useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import {
  agentError,
  applyAgentConfig,
  previewAgentPlan,
  type AgentClient,
} from "../../services/agents"
import type { AgentClientId } from "./agent-clients"

export type AgentGroupConfigFailure = {
  client: AgentClientId
  error: ReturnType<typeof agentError>
}

// 分组一键配置：按客户端去重后写入完整默认目录，逐项记录结果，避免一项失败中断整组。
export function useAgentGroupConfig(clients: AgentClientId[], items: AgentClient[]) {
  const queryClient = useQueryClient()
  const targets = [...new Set(clients)]
  const [busy, setBusy] = useState(false)
  const [done, setDone] = useState(0)
  const [failures, setFailures] = useState<AgentGroupConfigFailure[]>([])
  const [finished, setFinished] = useState<number | null>(null)

  async function run() {
    if (busy) return
    setBusy(true)
    setDone(0)
    setFailures([])
    setFinished(null)
    const failed: AgentGroupConfigFailure[] = []
    let succeeded = 0
    for (const client of targets) {
      const item = items.find((entry) => entry.client === client)
      if (!item) {
        failed.push({ client, error: "failed" })
      } else {
        try {
          // 未提供 modelIds 时后端写入完整目录，默认模型沿用目录建议值。
          const plan = await previewAgentPlan({ client, revision: item.revision })
          await applyAgentConfig({
            client,
            revision: item.revision,
            catalogRevision: plan.catalogRevision,
          })
          succeeded += 1
        } catch (error) {
          failed.push({ client, error: agentError(error) })
        }
      }
      setDone((value) => value + 1)
      setFailures([...failed])
    }
    setFinished(succeeded)
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["agent-clients"] }),
      queryClient.invalidateQueries({ queryKey: ["agent-configured-models"] }),
    ])
    setBusy(false)
  }

  return { busy, done, total: targets.length, failures, finished, run }
}
