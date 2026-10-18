import { useTranslation } from "react-i18next"
import type { LegacyEntry } from "../../services/legacy-import"

const states = {
  ready: "legacy.state.ready",
  duplicate: "legacy.state.duplicate",
  invalid: "legacy.state.invalid",
  login_required: "legacy.state.login",
  unsupported: "legacy.state.unsupported",
  billing_confirmation: "legacy.state.billing",
  endpoint_confirmation: "legacy.state.endpoint",
  credential_unavailable: "legacy.state.credential",
  imported: "legacy.state.imported",
  failed: "legacy.state.failed",
} as const

export function LegacyImportEntries({ entries }: { entries: LegacyEntry[] }) {
  const { t } = useTranslation()
  return (
    <ul
      className="max-h-64 space-y-2 overflow-y-auto rounded-lg border p-3"
      aria-label={t("legacy.entries")}
    >
      {entries.map((entry) => (
        <li key={entry.index} className="flex flex-wrap justify-between gap-x-3 gap-y-1">
          <span className="min-w-0 break-all">
            {entry.name} <span className="text-muted-foreground">({entry.provider})</span>
          </span>
          <span className="text-muted-foreground">
            {t(states[entry.state as keyof typeof states] ?? "legacy.state.invalid")}
            {!entry.enabled ? ` · ${t("legacy.disabled")}` : ""}
          </span>
        </li>
      ))}
    </ul>
  )
}
