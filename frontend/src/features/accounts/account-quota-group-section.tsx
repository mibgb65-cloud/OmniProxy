import type { ReactNode } from "react"
import { useTranslation } from "react-i18next"
import type { QuotaGroupSection } from "../google/google-quota-groups"
import { googleQuotaGroupLabel } from "../google/google-quota-model"

/**
 * 账号页的额度分区：标题给出该区的分组名和账号数，正文是该区的卡片网格或列表。
 * 未知分组单独说明原因，避免读者把「未获取」当成「没有该分组」。
 */
export function AccountQuotaGroupSection({
  section,
  children,
}: {
  section: QuotaGroupSection
  children: ReactNode
}) {
  const { t } = useTranslation()
  const title = section.names
    ? section.names.map((name) => googleQuotaGroupLabel(name, t)).join(" · ")
    : section.kind === "apiKey"
      ? t("accounts.quotaGroup.apiKey")
      : t("accounts.quotaGroup.unknown")
  return (
    <section aria-label={title} className="min-w-0 space-y-3">
      <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
        <h2 className="text-sm font-semibold wrap-anywhere">{title}</h2>
        <span className="text-muted-foreground text-xs">
          {t("accounts.count", { count: section.accounts.length })}
        </span>
      </div>
      {section.kind === "unknown" ? (
        <p className="text-muted-foreground text-xs leading-5">
          {t("accounts.quotaGroup.unknownHint")}
        </p>
      ) : null}
      {children}
    </section>
  )
}
