import { useEffect, useRef, useState } from "react"
import type { CodexActivationStatus } from "../../services/codex-settings"
import { CodexQuotaActivationToast } from "./codex-quota-activation-toast"
import { useCodexQuotaActivation } from "./use-codex-quota-activation"

// 真正检查过账号的轮次才提示；没有可检查账号的空轮次不打扰。
function worthNotice(status: CodexActivationStatus) {
  return status.checked > 0
}

// 结果相同就视为同一结果，即使完成时间变了也不重复弹出。
function resultKey(status: CodexActivationStatus) {
  return [
    status.checked,
    status.sent,
    status.activated,
    status.failed,
    status.pending,
    status.errorCode,
  ].join("|")
}

// 应用级监视器：不依赖设置面板是否打开，后台结果一到就提示。
export function CodexQuotaActivationWatcher() {
  const { enabled, status } = useCodexQuotaActivation()
  const seen = useRef("")
  const [notice, setNotice] = useState<{ key: string; status: CodexActivationStatus }>()
  const result = status.data
  useEffect(() => {
    if (!enabled || !result || result.running || !result.updatedAt) return
    if (!worthNotice(result)) return
    const key = resultKey(result)
    if (seen.current === key) return
    seen.current = key
    setNotice({ key, status: result })
  }, [enabled, result])
  return notice ? <CodexQuotaActivationToast key={notice.key} status={notice.status} /> : null
}
