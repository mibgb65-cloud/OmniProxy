import { Dialog } from "radix-ui"
import { useEffect, useRef, useState } from "react"
import type { ApplicationUpdateInstalled } from "../../services/updates"
import { useAppPreferences } from "../settings/app-preferences-state"
import { UpdateCompletedDialog } from "./update-completed-dialog"
import { useApplicationUpdate, useApplicationUpdateActions } from "./use-application-update"

/**
 * 应用级监视器：自我更新替换程序并重新打开后，后端会报告一次「刚刚升级到当前版本」，
 * 这里把它变成一次可见的「更新完成」浮窗。确认、关闭、Esc 与遮罩点击都算已读；
 * 同一会话内不再重复弹出，避免回执写入失败时反复打扰。
 */
export function UpdateCompletedWatcher() {
  const query = useApplicationUpdate()
  const { acknowledge } = useApplicationUpdateActions()
  const show = useAppPreferences((state) => state.show)
  const seen = useRef(new Set<string>())
  const [notice, setNotice] = useState<ApplicationUpdateInstalled>()
  const installed = query.data?.installed

  useEffect(() => {
    if (!installed || seen.current.has(installed.version)) return
    seen.current.add(installed.version)
    setNotice({ version: installed.version, from: installed.from })
  }, [installed])

  if (!notice) return null

  const dismiss = (openNotes: boolean) => {
    setNotice(undefined)
    acknowledge.mutate(undefined, { onSuccess: () => openNotes && show("about") })
  }

  return (
    <Dialog.Root
      open
      onOpenChange={(value) => {
        if (!value && notice) dismiss(false)
      }}
    >
      <UpdateCompletedDialog
        installed={notice}
        onConfirm={() => dismiss(false)}
        onOpenNotes={() => dismiss(true)}
      />
    </Dialog.Root>
  )
}
