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
