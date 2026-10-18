import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card"
import { Input } from "../../components/ui/input"
import { LegacyImportEntries } from "./legacy-import-entries"
import { useLegacyImport } from "./use-legacy-import"
import { useSelectDirectory } from "./use-storage"

export function LegacyImportCard() {
  const { t } = useTranslation()
  const [directory, setDirectory] = useState("")
  const { preview, apply } = useLegacyImport()
  const select = useSelectDirectory()
  const busy = preview.isPending || apply.isPending || select.isPending
  const changeDirectory = (path: string) => {
    setDirectory(path)
    preview.reset()
    apply.reset()
  }
  const error = preview.error ?? apply.error ?? select.error
  const previewData = preview.data
  return (
    <Card className="min-w-0 gap-4 rounded-2xl py-4 shadow-none">
      <CardHeader className="px-5">
        <CardTitle className="text-base">{t("legacy.title")}</CardTitle>
        <CardDescription>{t("legacy.description")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-3 px-5 text-sm" aria-busy={busy}>
        <p className="text-muted-foreground">{t("legacy.scope")}</p>
        <div className="space-y-2">
          <label htmlFor="legacy-directory">{t("legacy.directory")}</label>
          <Input
            id="legacy-directory"
            maxLength={4096}
            value={directory}
            disabled={busy}
            placeholder={t("legacy.automatic")}
            onChange={(event) => changeDirectory(event.target.value)}
          />
          <div className="flex flex-wrap gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={busy}
              onClick={() =>
                select.mutate(t("legacy.directory"), {
                  onSuccess: (path) => {
                    if (path) changeDirectory(path)
                  },
                })
              }
            >
              {t("settings.storage.browse")}
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={busy}
              onClick={() => {
                apply.reset()
                preview.mutate(directory.trim())
              }}
            >
              {preview.isPending ? t("legacy.reading") : t("legacy.preview")}
            </Button>
          </div>
        </div>
        {previewData ? (
          <>
            <p className="text-muted-foreground text-xs break-all">{previewData.directory}</p>
            <p role="status">
              {t("legacy.summary", {
                ready: previewData.ready,
                skipped: previewData.entries.length - previewData.ready,
              })}
            </p>
            <LegacyImportEntries entries={apply.data?.entries ?? previewData.entries} />
            <Button
              size="sm"
              disabled={busy || previewData.ready === 0 || apply.isSuccess}
              onClick={() =>
                apply.mutate({ directory: previewData.directory, snapshot: previewData.snapshot })
              }
            >
              {apply.isPending
                ? t("legacy.importing")
                : t("legacy.confirm", { count: previewData.ready })}
            </Button>
          </>
        ) : null}
        {apply.data ? (
          <p role="status">
            {t("legacy.result", { imported: apply.data.imported, failed: apply.data.failed })}
          </p>
        ) : null}
        {error ? (
          <p role="alert" className="text-destructive">
            {String(error.message).includes("source_changed")
              ? t("legacy.changed")
              : t("legacy.error")}
          </p>
        ) : null}
      </CardContent>
    </Card>
  )
}
