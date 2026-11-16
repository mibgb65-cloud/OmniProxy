import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  checkinDuMate,
  getDuMatePoints,
  importDuMateCookie,
  importDuMateDesktop,
  type AccountItem,
  type DuMateCheckin,
  type DuMateImportResult,
} from "../../services/omniproxy"

const pointsKey = (accountId: string) => ["dumate-points", accountId] as const

/** 配额页与账号页共用同一查询键，避免同一账号重复访问控制台；失败不重试。 */
export function duMatePointsOptions(accountId: string) {
  return queryOptions({
    queryKey: pointsKey(accountId),
    queryFn: () => getDuMatePoints(accountId),
    staleTime: 60_000,
    retry: false,
  })
}

/** 积分只在账号启用且列表需要时查询；失败不重试，按 state 展示。 */
export function useDuMatePoints(account: AccountItem, enabled = true) {
  return useQuery({
    ...duMatePointsOptions(account.id),
    enabled: enabled && account.enabled,
  })
}

/** 导入成功会新增或更新账号，因此同时失效账号与模型目录缓存。 */
export function useDuMateImport(providerId: string) {
  const client = useQueryClient()
  const refresh = () =>
    Promise.all([
      client.invalidateQueries({ queryKey: ["accounts"] }),
      client.invalidateQueries({ queryKey: ["model-catalog"] }),
      client.invalidateQueries({ queryKey: ["default-routes"] }),
    ])
  const desktop = useMutation({
    mutationFn: () => importDuMateDesktop(providerId),
    onSuccess: refresh,
  })
  const cookie = useMutation({
    mutationFn: (raw: string) => importDuMateCookie(providerId, raw),
    onSuccess: refresh,
  })
  return { desktop, cookie }
}

/** 签到后积分与账号状态都可能变化，两类查询一并失效。 */
export function useDuMateCheckin(account: AccountItem) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: () => checkinDuMate(account.id),
    onSettled: () =>
      Promise.all([
        client.invalidateQueries({ queryKey: pointsKey(account.id) }),
        client.invalidateQueries({ queryKey: ["accounts"] }),
      ]),
  })
}

const CHECKIN_KEYS = {
  claimed: "dumate.points.checkinClaimed",
  done: "dumate.points.checkinDone",
  pending: "dumate.points.checkinPending",
  failed: "dumate.points.checkinFailed",
} as const

/** 导入结果的成功文案区分新建与更新同一身份。 */
export function importSuccessKey(result: DuMateImportResult) {
  return result.mode === "updated" ? "dumate.import.updated" : "dumate.import.created"
}

/** 签到结果只按 outcome 映射文案；积分变动交给通用格式化。 */
export function checkinMessageKey(result: DuMateCheckin) {
  return CHECKIN_KEYS[result.outcome as keyof typeof CHECKIN_KEYS] ?? CHECKIN_KEYS.failed
}
