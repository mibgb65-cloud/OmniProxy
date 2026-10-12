import { useEffect, useRef, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import type { AccountItem } from "../../services/omniproxy"
import {
  consumeSubscriptionReset,
  readSubscriptionResetStatus,
  type SubscriptionResetInput,
} from "../../services/subscription"
import { subscriptionQuotaKey, useSubscriptionQuota } from "./use-subscription-quota"
import { resetRequestID } from "./reset-request-id"

export function useResetCredits(account: AccountItem) {
  const client = useQueryClient()
  const quota = useSubscriptionQuota(account)
  const status = useQuery({
    queryKey: ["subscription-reset-status", account.id],
    queryFn: () => readSubscriptionResetStatus(account.id),
    retry: false,
    staleTime: 0,
  })
  const [selection, setSelection] = useState<string | null>(null)
  const sending = useRef(false)
  const mutation = useMutation({
    mutationKey: ["subscription-reset", account.id],
    mutationFn: (input: SubscriptionResetInput) => consumeSubscriptionReset(input),
    retry: false,
    onSettled: async (result) => {
      if (result?.quota) client.setQueryData(subscriptionQuotaKey(account), result.quota)
      else await client.invalidateQueries({ queryKey: ["subscription-quota", account.id] })
      await client.invalidateQueries({ queryKey: ["subscription-reset-status", account.id] })
    },
  })
  const refetchQuota = quota.refetch
  // 弹窗打开时核对最新数量，卡片入口继续复用同一缓存。
  useEffect(() => {
    void refetchQuota({ cancelRefetch: false })
  }, [refetchQuota])
  const pending =
    mutation.data?.operation.state === "pending"
      ? mutation.data.operation
      : status.data?.operationId && status.data.state === "pending"
        ? status.data
        : undefined
  const submit = async (retry = false) => {
    if (sending.current || (!retry && selection === null)) return
    sending.current = true
    try {
      await mutation.mutateAsync({
        accountId: account.id,
        creditId: retry ? pending?.creditId || "" : selection || "",
        operationId: retry ? pending?.operationId || "" : resetRequestID(),
      })
      setSelection(null)
    } catch {
      // 服务端先落库后调用官方接口；IPC 失败后重新读取，恢复可能已经提交的请求。
      await client.invalidateQueries({ queryKey: ["subscription-reset-status", account.id] })
    } finally {
      sending.current = false
    }
  }
  const receipt =
    mutation.data?.operation.state === "completed"
      ? mutation.data.operation
      : status.data?.state === "completed"
        ? status.data
        : undefined
  return { quota, status, mutation, selection, setSelection, pending, receipt, submit }
}
