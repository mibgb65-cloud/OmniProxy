/** Byte-size formatting for storage displays. Units follow the IEC-style
 * 1024 progression with short suffixes, matching what users expect from
 * file managers. */

const UNITS = ["B", "KB", "MB", "GB", "TB"] as const

export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return "0 B"
  }
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < UNITS.length - 1) {
    value /= 1024
    unit += 1
  }
  const digits = value < 10 && unit > 0 ? 1 : 0
  return `${value.toFixed(digits)} ${UNITS[unit]}`
}
