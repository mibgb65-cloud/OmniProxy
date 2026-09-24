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

/** Formats a duration as HH:MM:SS (language-neutral). */
export function formatDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) {
    ms = 0
  }
  const totalSeconds = Math.floor(ms / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  const pad = (n: number) => String(n).padStart(2, "0")
  return `${pad(hours)}:${pad(minutes)}:${pad(seconds)}`
}
