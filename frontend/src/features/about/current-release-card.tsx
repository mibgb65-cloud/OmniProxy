import { FileText } from "lucide-react"
import { useId } from "react"
import { useTranslation } from "react-i18next"
import { Markdown } from "../../components/common/markdown"
import { Card } from "../../components/ui/card"
import { getCurrentReleaseNotes, withoutTopHeading } from "./current-release"

export function CurrentReleaseCard({
  version,
  pending,
  failed,
}: {
  version: string | undefined
  pending: boolean
  failed: boolean
}) {
  const { t } = useTranslation()
  const titleId = useId()
  const notes = getCurrentReleaseNotes(version)
  return (
    <Card role="region" aria-labelledby={titleId} className="gap-0 py-0 shadow-none">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b px-4 py-3.5">
        <h3 id={titleId} className="flex items-center gap-2 text-sm font-medium">
          <FileText className="text-muted-foreground size-4" aria-hidden="true" />
          {t("updates.currentNotes")}
        </h3>
        {version ? (
          <span className="text-muted-foreground font-mono text-xs">{version}</span>
        ) : null}
      </div>
      <div className="p-4" aria-busy={pending}>
        {pending || failed || !notes ? (
          <p className="text-muted-foreground text-sm" role={failed ? "alert" : undefined}>
            {t(
              pending
                ? "updates.notesLoading"
                : failed
                  ? "updates.versionFailed"
                  : "updates.noCurrentNotes",
            )}
          </p>
        ) : (
          <Markdown source={withoutTopHeading(notes)} />
        )}
      </div>
    </Card>
  )
}
