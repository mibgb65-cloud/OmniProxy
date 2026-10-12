import { useEffect, useRef, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import { getLogHead } from "../../services/logs"
import { logQueryKey, mergeLogData, readLogRange, type LogData } from "./log-stream"

export function useLiveLogs() {
  const client = useQueryClient()
  const [paused, setPaused] = useState(false)
  const [frozen, setFrozen] = useState<LogData | undefined>()
  const [latestNotice, setLatestNotice] = useState(0)
  const pausedRef = useRef(paused)
  const query = useQuery({
    queryKey: logQueryKey,
    enabled: false,
    gcTime: Infinity,
    staleTime: Infinity,
    retry: false,
    queryFn: async ({ signal }) => {
      const previous = client.getQueryData<LogData>(logQueryKey)
      const snapshot = await readLogRange(previous?.cursor ?? 0, signal)
      if (signal.aborted) throw new DOMException("Aborted", "AbortError")
      return mergeLogData(client.getQueryData<LogData>(logQueryKey), snapshot)
    },
  })
  const { refetch, data: queryData, isError } = query

  // 先订阅再读取；快照游标与事件序号共同补齐读取期间出现的新记录。
  useEffect(() => {
    const off = Events.On("logs:updated", (event) => {
      const latest = event.data.latestId
      if (!Number.isSafeInteger(latest) || latest < 0) return
      setLatestNotice(latest)
      if (!pausedRef.current) void refetch({ cancelRefetch: false })
    })
    void refetch({ cancelRefetch: false })
    return off
  }, [refetch])

  useEffect(() => {
    if (!paused && queryData && latestNotice > queryData.cursor && !isError) {
      void refetch({ cancelRefetch: false })
    }
  }, [paused, latestNotice, queryData, isError, refetch])

  const clear = useMutation({
    mutationFn: getLogHead,
    onSuccess: (head) => {
      const current = client.getQueryData<LogData>(logQueryKey)
      if (!current) return
      const cleared = {
        ...current,
        entries: [],
        cursor: head,
        latestId: head,
        clearAfter: head,
        dropped: false,
      }
      client.setQueryData(logQueryKey, cleared)
      if (pausedRef.current) setFrozen(cleared)
      setLatestNotice(head)
    },
  })

  function togglePaused() {
    pausedRef.current = !pausedRef.current
    if (paused) {
      setFrozen(undefined)
      setPaused(false)
      void refetch({ cancelRefetch: false })
    } else {
      setFrozen(query.data)
      setPaused(true)
    }
  }
  const data = paused ? frozen : query.data
  return {
    data,
    query,
    paused,
    togglePaused,
    clear,
    pending: Math.max(0, latestNotice - (data?.cursor ?? 0)),
    resume: () => {
      pausedRef.current = false
      setFrozen(undefined)
      setPaused(false)
      void query.refetch({ cancelRefetch: false })
    },
  }
}
