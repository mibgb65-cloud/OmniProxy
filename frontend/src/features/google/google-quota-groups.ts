import type { AccountItem } from "../../services/omniproxy"
import type { GoogleSubscriptionQuota } from "../../services/google-subscription"

/**
 * 账号的额度分组签名：官方返回的分组名去重后的列表，保持上游顺序（卡片内也按此顺序展示）。
 * 签名只由分组名构成，不含比例与重置时间，因此额度用掉多少不会改变账号所属分区。
 * null 表示没有可用分组（未登录、尚未刷新、读取失败或上游未返回分组名）。
 */
export type QuotaGroupSignature = string[] | null

export type QuotaGroupSectionKind = "groups" | "apiKey" | "unknown"

export type QuotaGroupSection = {
  /** 分区稳定键，可直接用作 React key；分组集合相同即同一键。 */
  key: string
  /** 分区归类原因，决定标题文案与提示。 */
  kind: QuotaGroupSectionKind
  /** 分区内的分组名；只有 `groups` 分区有值。 */
  names: QuotaGroupSignature
  accounts: AccountItem[]
}

/** 分区顺序：已知分组的账号在前（分组更多者优先），API Key 账号其次，未知最后。 */
const sectionRank: Record<QuotaGroupSectionKind, number> = { groups: 0, apiKey: 1, unknown: 2 }

function compareText(left: string, right: string): number {
  if (left === right) return 0
  return left < right ? -1 : 1
}

export function quotaGroupSignature(quota?: GoogleSubscriptionQuota): QuotaGroupSignature {
  const names: string[] = []
  for (const group of quota?.groups ?? []) {
    const name = (group.name ?? "").trim()
    if (name !== "" && !names.includes(name)) names.push(name)
  }
  return names.length ? names : null
}

/**
 * 按额度分组集合把账号聚合成分区：分组集合相同的账号相邻，分组更多的账号单独成区。
 * 只按事实归类——不读套餐档位，也不推断分组名之外的含义。
 */
export function buildQuotaGroupSections(
  accounts: AccountItem[],
  quotas: ReadonlyMap<string, GoogleSubscriptionQuota | undefined>,
): QuotaGroupSection[] {
  const sections = new Map<string, QuotaGroupSection>()
  for (const account of accounts) {
    const subscription = account.authMethod === "subscription_login"
    const names = subscription ? quotaGroupSignature(quotas.get(account.id)) : null
    const kind: QuotaGroupSectionKind = names ? "groups" : subscription ? "unknown" : "apiKey"
    const key = names ? `groups:${[...names].sort(compareText).join("\u0000")}` : kind
    const section = sections.get(key) ?? { key, kind, names, accounts: [] }
    section.accounts.push(account)
    sections.set(key, section)
  }
  return [...sections.values()].sort(compareSections)
}

function compareSections(left: QuotaGroupSection, right: QuotaGroupSection): number {
  if (left.kind !== right.kind) return sectionRank[left.kind] - sectionRank[right.kind]
  const bySize = (right.names?.length ?? 0) - (left.names?.length ?? 0)
  if (bySize !== 0) return bySize
  return compareText(left.key, right.key)
}
