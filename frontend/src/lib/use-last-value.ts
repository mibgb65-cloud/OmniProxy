import { useState } from "react"

/**
 * 记住最后一个非空值。弹窗关闭时父组件会清空数据，但退出动画期间仍需要内容可渲染，
 * 因此保留上一次的值；真正卸载后自然释放。
 * 采用 React 官方的「渲染期间根据 props 调整 state」写法，值只在变化时更新一次。
 */
export function useLastValue<T>(value: T | null | undefined): T | undefined {
  const [last, setLast] = useState<T>()
  if (value && value !== last) setLast(value)
  return value ?? last
}
