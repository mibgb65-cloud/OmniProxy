import { useEffect, useRef, useState } from "react"

/**
 * 每次打开都返回一个新的 key。弹窗内容改为常驻以支持退出动画后，
 * 用它在重新打开时换掉旧实例，避免上次的草稿与临时状态残留。
 */
export function useOpenGeneration(open: boolean) {
  const [generation, setGeneration] = useState(0)
  const wasOpen = useRef(false)
  useEffect(() => {
    if (open && !wasOpen.current) setGeneration((value) => value + 1)
    wasOpen.current = open
  }, [open])
  return generation
}
