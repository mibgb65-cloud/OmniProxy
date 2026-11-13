/** 积分最多两位小数；整数不带多余尾零。 */
export function formatPoints(value: number): string {
  if (!Number.isFinite(value)) return "0"
  return Number.isInteger(value) ? String(value) : value.toFixed(2)
}
