import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { Tooltip } from "../../components/ui/tooltip"
import type { AccountItem } from "../../services/omniproxy"
import { useAccountTokenUsage } from "./use-account-token-usage"

// 连续请求之间可能只隔几百毫秒；观测到进行中后多保留一会儿，避免圆环闪断。
const HOLD_MS = 1200

// 只有展示本机 Token 用量的账号（River API Key）才有进行中状态可读。
function tracksPendingRequests(account: AccountItem) {
  return account.authMethod !== "subscription_login" && account.providerId === "builtin_river"
}

export function AccountInUseIndicator({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const tracked = tracksPendingRequests(account)
  const { data } = useAccountTokenUsage(account, { enabled: tracked })
  const pending = tracked && (data?.pendingRequests ?? 0) > 0
  const [visible, setVisible] = useState(pending)

  useEffect(() => {
    // 观测到进行中立即显示；否则多保留 HOLD_MS 再隐藏，避免连续请求之间闪断。
    const timer = setTimeout(() => setVisible(pending), pending ? 0 : HOLD_MS)
    return () => clearTimeout(timer)
  }, [pending])

  // 固定 16px 槽位：圆环出现和消失都不挪动旁边的启用徽标。
  if (!tracked) return null
  const label = t("accounts.inUse")
  return (
    <span className="flex size-4 shrink-0 items-center justify-center">
      {visible ? (
        <Tooltip content={label}>
          <span
            role="status"
            aria-label={label}
            className="border-primary/25 border-t-primary size-3.5 animate-spin rounded-full border-2 motion-reduce:animate-none"
          />
        </Tooltip>
      ) : null}
    </span>
  )
}
