import { useTranslation } from "react-i18next"

export function AccountStatusBadge({ enabled }: { enabled: boolean }) {
  const { t } = useTranslation()
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs whitespace-nowrap ${
        enabled
          ? "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-300"
          : "border-border bg-muted/50 text-muted-foreground"
      }`}
    >
      <span
        className={`size-1.5 rounded-full ${enabled ? "bg-emerald-500" : "bg-muted-foreground"}`}
        aria-hidden="true"
      />
      {enabled ? t("accounts.enabled") : t("accounts.disabled")}
    </span>
  )
}
