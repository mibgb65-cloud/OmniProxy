import { useIsMutating, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useRef } from "react"
import {
  getAgentInstallations,
  refreshAgentInstallations,
  installAgent,
  checkAgentUpdate,
  openAgentInstallationOfficial,
  openClaudeCode,
  openAgentClient,
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
  const open = useMutation({ mutationFn: (id: AgentInstallTarget) => openAgentClient(id) })
  const update = useMutation({
    mutationFn: (id: AgentInstallTarget) => checkAgentUpdate(id),
    onSuccess: (result, id) => {
      if (!result.currentVersion) return
      client.setQueryData<Awaited<ReturnType<typeof getAgentInstallations>>>(
        agentInstallQueryKey,
        (snapshot) =>
          snapshot && {
            ...snapshot,
            items: (snapshot.items ?? []).map((item) =>
              item.id === id
                ? { ...item, detection: { ...item.detection, version: result.currentVersion } }
                : item,
            ),
          },
      )
    },
  })
  const mutating = useIsMutating({ mutationKey: agentInstallQueryKey }) > 0
  return {
    data,
    refresh,
    install,
    official,
    launch,
    open,
    update,
    busy: !!data.data?.busy || mutating,
    start: (id: AgentInstallTarget) => {
      refresh.reset()
      official.reset()
      launch.reset()
      open.reset()
      install.mutate(id)
    },
    recheck: () => {
      update.reset()
      install.reset()
      official.reset()
      launch.reset()
      open.reset()
      refresh.mutate()
    },
  }
}

export type AgentInstallations = ReturnType<typeof useAgentInstallations>
