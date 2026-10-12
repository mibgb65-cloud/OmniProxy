import { useIsMutating, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useRef } from "react"
import {
  getAgentInstallations,
  refreshAgentInstallations,
  installAgent,
  openAgentInstallationOfficial,
  openClaudeCode,
} from "../../services/agent-installation"
import type { AgentInstallTarget } from "./agent-installation-targets"

export const agentInstallQueryKey = ["agent-installations"] as const

export function useAgentInstallations(active: boolean) {
  const client = useQueryClient()
  const data = useQuery({
    queryKey: agentInstallQueryKey,
    queryFn: getAgentInstallations,
    enabled: active,
    refetchInterval: (query) => (query.state.data?.busy ? 1000 : false),
  })
  const wasBusy = useRef(false)
  useEffect(() => {
    if (wasBusy.current && data.data && !data.data.busy) {
      void client.invalidateQueries({ queryKey: ["agent-clients"] })
      void client.invalidateQueries({ queryKey: ["gateway-runtime"] })
    }
    wasBusy.current = data.data?.busy ?? false
  }, [data.data, client])
  const refresh = useMutation({
    mutationKey: [...agentInstallQueryKey, "refresh"],
    mutationFn: refreshAgentInstallations,
    onSuccess: (snapshot) => client.setQueryData(agentInstallQueryKey, snapshot),
  })
  const install = useMutation({
    mutationKey: [...agentInstallQueryKey, "install"],
    mutationFn: (id: AgentInstallTarget) => installAgent(id),
    onSuccess: (snapshot) => client.setQueryData(agentInstallQueryKey, snapshot),
    onError: () => client.invalidateQueries({ queryKey: agentInstallQueryKey }),
  })
  const official = useMutation({
    mutationFn: (id: AgentInstallTarget) => openAgentInstallationOfficial(id),
  })
  const launch = useMutation({ mutationFn: openClaudeCode })
  const mutating = useIsMutating({ mutationKey: agentInstallQueryKey }) > 0
  return {
    data,
    refresh,
    install,
    official,
    launch,
    busy: !!data.data?.busy || mutating,
    start: (id: AgentInstallTarget) => {
      refresh.reset()
      official.reset()
      launch.reset()
      install.mutate(id)
    },
    recheck: () => {
      install.reset()
      official.reset()
      launch.reset()
      refresh.mutate()
    },
  }
}

export type AgentInstallations = ReturnType<typeof useAgentInstallations>
