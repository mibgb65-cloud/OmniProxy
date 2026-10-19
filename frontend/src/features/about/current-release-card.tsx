import { FileText } from "lucide-react"
import { useId } from "react"
import { useTranslation } from "react-i18next"
import { Card } from "../../components/ui/card"
import { getCurrentReleaseNotes, parseReleaseNotes } from "./current-release"

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
      <div
        className="space-y-3 p-4 text-sm leading-relaxed [overflow-wrap:anywhere]"
        aria-busy={pending}
      >
        {pending || failed || !notes ? (
          <p className="text-muted-foreground" role={failed ? "alert" : undefined}>
            {t(
              pending
                ? "updates.notesLoading"
                : failed
                  ? "updates.versionFailed"
                  : "updates.noCurrentNotes",
            )}
          </p>
        ) : (
          parseReleaseNotes(notes).map((block, index) =>
            block.kind === "list" ? (
              <ul key={index} className="text-muted-foreground list-disc space-y-2 pl-5">
                {block.items.map((item, itemIndex) => (
                  <li key={itemIndex}>{item}</li>
                ))}
              </ul>
            ) : block.kind === "heading" ? (
              <h4 key={index} className="pt-2 font-medium">
                {block.text}
              </h4>
            ) : (
              <p key={index} className="text-muted-foreground whitespace-pre-wrap">
                {block.text}
              </p>
            ),
          )
        )}
      </div>
    </Card>
  )
}
