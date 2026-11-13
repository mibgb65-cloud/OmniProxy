/**
 * 图表通用数值工具。
 * Token 合计会超出 Number 安全范围，坐标轴与比例全程使用 BigInt，只在绘制时转成 Number。
 */

/** 纵轴上限：留 5% 余量后按一位有效数字向上取整，让刻度落在 0／400／800／1.2K 这类整数上。 */
export function chartAxisMaximum(maximum: bigint) {
  if (maximum <= 0n) return 4n
  const headroom = (maximum * 105n + 99n) / 100n
  const quarter = (headroom + 3n) / 4n
  if (quarter <= 1n) return 4n
  const magnitude = 10n ** BigInt(quarter.toString().length - 1)
  const step = quarter % magnitude === 0n ? quarter : (quarter / magnitude + 1n) * magnitude
  return step * 4n
}

/** 纵轴从上到下的五个刻度值，首项为上限、末项为 0。 */
export function chartAxisTicks(maximum: bigint) {
  const axis = chartAxisMaximum(maximum)
  return [axis, (axis * 3n) / 4n, axis / 2n, axis / 4n, 0n]
}

/** 横轴标签抽稀间隔：最多 6 个标签，末位标签始终显示。 */
export function chartLabelStep(count: number) {
  return count <= 1 ? 1 : Math.max(1, Math.ceil((count - 1) / 5))
}

/** 占比百分比，保留两位小数；分母为 0 时返回 0。 */
export function chartPercent(value: bigint, total: bigint) {
  return total > 0n ? Number((value * 10000n) / total) / 100 : 0
}

export function chartCompact(value: bigint, language: string) {
  return new Intl.NumberFormat(language === "zh" ? "zh-CN" : "en-US", {
    notation: "compact",
    maximumFractionDigits: 1,
  }).format(value)
}

/** 毫秒时长：小于 1 秒用 ms，否则用秒；未知值保持 —，不从其他数字推测。 */
export function chartDuration(value: number | null | undefined, language: string) {
  if (value === null || value === undefined || !Number.isFinite(value)) return "—"
  const locale = language === "zh" ? "zh-CN" : "en-US"
  return value < 1000
    ? `${new Intl.NumberFormat(locale, { maximumFractionDigits: 0 }).format(value)} ms`
    : `${new Intl.NumberFormat(locale, { maximumFractionDigits: 2 }).format(value / 1000)} s`
}
