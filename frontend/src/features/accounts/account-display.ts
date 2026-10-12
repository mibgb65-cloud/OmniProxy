import type { AccountItem } from "../../services/omniproxy"

export function accountDisplayName(account: AccountItem): string {
  return account.authMethod === "subscription_login" && account.authEmail
    ? account.authEmail
    : account.name
}

export function formatAccountTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "—"
  const pad = (part: number) => String(part).padStart(2, "0")
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}
