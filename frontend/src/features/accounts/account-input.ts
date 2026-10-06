export const MAX_BATCH_KEYS = 100
export const MAX_KEY_BYTES = 8192
export const MAX_BATCH_TEXT_BYTES = MAX_BATCH_KEYS * (MAX_KEY_BYTES + 2)

export function validAccountName(value: string): boolean {
  const name = value.trim()
  return (
    Array.from(name).length <= 80 &&
    new TextEncoder().encode(name).length <= 256 &&
    !/[\p{Cc}]/u.test(name)
  )
}

export function validAPIKey(value: string): boolean {
  const key = value.trim()
  return (
    key.length > 0 &&
    new TextEncoder().encode(key).length <= MAX_KEY_BYTES &&
    !/[\p{Cc}]/u.test(key)
  )
}

export function splitAPIKeys(value: string): string[] {
  return value
    .split(/\r\n|\n|\r/u)
    .map((key) => key.trim())
    .filter(Boolean)
}
