const minimumFeedbackMs = 650

// 只补足成功动作的视觉反馈时间；请求和缓存更新先执行，失败立即向上传递。
export async function withGatewayFeedback<T>(operation: () => Promise<T>): Promise<T> {
  const startedAt = Date.now()
  const result = await operation()
  const remaining = minimumFeedbackMs - (Date.now() - startedAt)
  if (remaining > 0) await new Promise<void>((resolve) => setTimeout(resolve, remaining))
  return result
}
