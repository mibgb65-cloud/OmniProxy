const stateKeys = {
  success: "activity.success",
  failed: "activity.failedState",
  cancelled: "activity.cancelled",
  pending: "activity.pending",
} as const
export function stateLabel(state: string) {
  switch (state) {
    case "success":
    case "failed":
    case "cancelled":
    case "pending":
      return stateKeys[state]
    default:
      return "activity.failedState"
  }
}
export function requestTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "—"
  const part = (n: number) => String(n).padStart(2, "0")
  return `${date.getFullYear()}-${part(date.getMonth() + 1)}-${part(date.getDate())} ${part(date.getHours())}:${part(date.getMinutes())}:${part(date.getSeconds())}`
}

/** Token 数量以十进制字符串传输，转数字前先确认仍在安全整数范围内。 */
function tokenCount(value: string) {
  try {
    const count = Number(BigInt(value))
    return Number.isSafeInteger(count) && count >= 0 ? count : null
  } catch {
    return null
  }
}

/**
 * 缓存命中率：上游报告的前缀缓存命中量占输入 Token 的比例。
 * 未报告（含历史记录）保持未知，真实的零命中显示 0%。
 */
export function formatCachedHit(
  cached: string,
  cachedKnown: boolean,
  input: string,
  inputKnown: boolean,
) {
  if (!cachedKnown || !inputKnown) return "—"
  const hits = tokenCount(cached)
  const total = tokenCount(input)
  if (hits === null || total === null || total <= 0) return "—"
  const percent = (hits / total) * 100
  if (percent >= 10) return `${Math.round(percent)}%`
  return `${Math.round(percent * 10) / 10}%`
}

function costText(value: number) {
  if (value > 0 && value < 0.000001) return "<$0.000001"
  const digits = value >= 1 ? 2 : value >= 0.01 ? 4 : 6
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: digits,
  }).format(value)
}

/** 成本是基础参考价估算值，不是账单；缓存明细未知时按未命中／全部命中给出区间。 */
export function formatCost(min: number, max: number, known: boolean) {
  if (!known || !Number.isFinite(min) || !Number.isFinite(max)) return "—"
  const low = costText(Math.min(min, max))
  const high = costText(Math.max(min, max))
  return low === high ? low : `${low}–${high}`
}

/**
 * 速度：输出 Token ÷ 生成耗时。流式请求扣除首字等待，只统计已结束且已报告输出 Token 的请求。
 */
export function formatSpeed(item: {
  outputTokens: string
  outputKnown: boolean
  latencyMs: number
  firstTokenMs?: number | null
  state: string
}) {
  if (item.state === "pending" || !item.outputKnown) return "—"
  const output = tokenCount(item.outputTokens)
  if (output === null || output <= 0 || item.latencyMs <= 0) return "—"
  const first = item.firstTokenMs ?? null
  const generation =
    first !== null && first > 0 && first < item.latencyMs ? item.latencyMs - first : item.latencyMs
  if (generation <= 0) return "—"
  const speed = output / (generation / 1000)
  if (!Number.isFinite(speed) || speed <= 0) return "—"
  return `≈${speed >= 100 ? Math.round(speed) : Math.round(speed * 10) / 10} tok/s`
}

/** 首字耗时只来自流式采集；非流式、历史记录和未产出内容的失败保持未知。 */
export function formatFirstToken(firstTokenMs?: number | null, state = "success") {
  if (firstTokenMs === null || firstTokenMs === undefined || state === "pending") return "—"
  return `${firstTokenMs} ms`
}
