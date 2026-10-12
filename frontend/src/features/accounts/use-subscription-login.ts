import { useEffect, useRef, useState } from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import {
  beginSubscription,
  cancelSubscription,
  openSubscription,
  readSubscription,
  currentSubscription,
  subscriptionError,
  type SubscriptionInput,
  type SubscriptionSession,
} from "../../services/subscription"

export function useSubscriptionLogin(scope: { providerId: string; accountId?: string }) {
  const client = useQueryClient()
  const [session, setSession] = useState<SubscriptionSession>()
  const [starting, setStarting] = useState(false)
  const [error, setError] = useState("")
  const alive = useRef(true)
  const pending = useRef(false)
  const resume = useQuery({
    queryKey: ["subscription-current", scope.providerId, scope.accountId],
    queryFn: () => currentSubscription(scope.providerId, scope.accountId ?? ""),
    staleTime: 0,
    gcTime: 0,
    retry: false,
  })
  const active = session ?? (resume.data?.id ? resume.data : undefined)
  const query = useQuery({
    queryKey: ["subscription-session", active?.id],
    queryFn: () => readSubscription(active!.id),
    enabled: !!active?.id && active.state === "waiting",
    refetchOnWindowFocus: false,
    retry: false,
    gcTime: 0,
  })
  const view = query.data ?? active
  useEffect(() => {
    alive.current = true
    return () => {
      alive.current = false
      // 浏览器授权独立于弹窗；关闭／切换页面保留回调，由显式取消或后台超时清理。
    }
  }, [])
  useEffect(() => {
    if (!active?.id) return
    const off = Events.On("subscription:updated", () => {
      void client.invalidateQueries({ queryKey: ["subscription-session", active.id] })
    })
    // 订阅事件后再读取一次，覆盖启动返回与监听建立之间的完成通知。
    void client.invalidateQueries({ queryKey: ["subscription-session", active.id] })
    return off
  }, [client, active?.id])
  useEffect(() => {
    if (view?.state === "completed") void client.invalidateQueries({ queryKey: ["accounts"] })
  }, [client, view?.state])
  const start = async (input: SubscriptionInput) => {
    if (pending.current) return
    pending.current = true
    setStarting(true)
    setError("")
    try {
      const next = await beginSubscription(input)
      if (alive.current) setSession(next)
      try {
        await openSubscription(next.id)
      } catch (cause) {
        if (alive.current) setError(subscriptionError(cause))
      }
    } catch (cause) {
      if (alive.current) setError(subscriptionError(cause))
    } finally {
      pending.current = false
      if (alive.current) setStarting(false)
    }
  }
  const open = async () => {
    if (!view) return
    try {
      await openSubscription(view.id)
      setError("")
    } catch (cause) {
      setError(subscriptionError(cause))
    }
  }
  const cancel = async () => {
    if (!view) return
    try {
      await cancelSubscription(view.id)
      await client.invalidateQueries({ queryKey: ["subscription-session", view.id] })
    } catch (cause) {
      setError(subscriptionError(cause))
    }
  }
  return {
    view,
    starting: starting || resume.isPending,
    error:
      error ||
      (resume.isError
        ? subscriptionError(resume.error)
        : query.isError
          ? "operation_failed"
          : view?.errorCode),
    start,
    open,
    cancel,
  }
}
