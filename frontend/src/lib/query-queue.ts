/** 限制实际 RPC 并发；离开页面时移除尚未开始的查询，避免后台持续读取账号。 */
export function createQueryQueue(concurrency: number) {
  if (!Number.isInteger(concurrency) || concurrency < 1) throw new RangeError("Invalid concurrency")
  let active = 0
  const waiting: Array<() => void> = []

  function drain() {
    while (active < concurrency && waiting.length > 0) waiting.shift()?.()
  }

  return function run<T>(operation: () => Promise<T>, signal?: AbortSignal): Promise<T> {
    return new Promise((resolve, reject) => {
      const abort = () => {
        const index = waiting.indexOf(start)
        if (index >= 0) waiting.splice(index, 1)
        reject(signal?.reason ?? new DOMException("Query cancelled", "AbortError"))
      }
      const start = () => {
        signal?.removeEventListener("abort", abort)
        if (signal?.aborted) {
          abort()
          return
        }
        active++
        // 已发出的 Wails 查询由后端超时结束，不能提前释放名额造成实际并发失控。
        void Promise.resolve()
          .then(operation)
          .then(resolve, reject)
          .finally(() => {
            active--
            drain()
          })
      }
      if (signal?.aborted) {
        abort()
        return
      }
      waiting.push(start)
      signal?.addEventListener("abort", abort, { once: true })
      drain()
    })
  }
}

export const queueSubscriptionQuery = createQueryQueue(1)
export const queueBalanceQuery = createQueryQueue(2)
