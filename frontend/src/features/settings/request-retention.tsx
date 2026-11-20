import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Database, Save, Trash2 } from "lucide-react"
import { useId, useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card"
import { Input } from "../../components/ui/input"
import { Switch } from "../../components/ui/switch"
import { formatBytes } from "../activity/activity-format"
import {
  clearRequestContent,
  readRetention,
  setRequestContentRecording,
  setRetention,
} from "../../services/activity"

export function RequestRetentionSettings() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const contentToggleId = useId()
  const [edited, setEdited] = useState<string | null>(null)
  const query = useQuery({ queryKey: ["request-retention"], queryFn: readRetention })
  const value = edited ?? String(query.data?.maxRecords ?? 10000)
  const number = Number(value)
  const valid =
    /^\d+$/.test(value) && Number.isSafeInteger(number) && number >= 100 && number <= 1000000
  const save = useMutation({
    mutationFn: () => setRetention(number),
    onSuccess: async (data) => {
      client.setQueryData(["request-retention"], data)
      setEdited(null)
      await client.invalidateQueries({ queryKey: ["requests"] })
    },
  })
  const contentToggle = useMutation({
    mutationFn: (enabled: boolean) => setRequestContentRecording(enabled),
    onSuccess: async (data) => {
      client.setQueryData(["request-retention"], data)
      await client.invalidateQueries({ queryKey: ["request-content"] })
    },
  })
  const contentClear = useMutation({
    mutationFn: clearRequestContent,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["request-content"] })
    },
  })
  return (
    <Card className="min-w-0 gap-4 rounded-2xl py-4 shadow-none">
      <CardHeader className="px-5">
        <CardTitle className="flex items-center gap-2 text-base">
          <Database className="size-4" aria-hidden="true" />
          {t("activity.retentionTitle")}
        </CardTitle>
        <CardDescription>{t("activity.retentionDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4 px-5">
        <div className="flex flex-wrap items-end gap-3">
          <div className="min-w-0 flex-1 space-y-2">
            <label htmlFor="request-record-limit" className="text-sm font-medium">
              {t("activity.retentionLabel")}
            </label>
            <Input
              id="request-record-limit"
              type="number"
              inputMode="numeric"
              min={100}
              max={1000000}
              step={100}
              value={value}
              disabled={!query.isSuccess || save.isPending}
              aria-invalid={!valid}
              aria-describedby="request-retention-hint"
              onChange={(e) => {
                setEdited(e.target.value)
                save.reset()
              }}
            />
          </div>
          <Button
            disabled={
              !query.isSuccess || !valid || save.isPending || number === query.data?.maxRecords
            }
            onClick={() => save.mutate()}
          >
            <Save aria-hidden="true" />
            {t(save.isPending ? "activity.saving" : "activity.save")}
          </Button>
        </div>
        <p id="request-retention-hint" className="text-muted-foreground text-xs leading-5">
          {t("activity.retentionHint")}
        </p>
        <div className="text-muted-foreground flex flex-wrap justify-between gap-2 text-xs">
          <span>{t("activity.logLimit")}</span>
          <span>{t("activity.stored", { count: query.data?.storedRecords ?? "—" })}</span>
        </div>
        {!valid ? (
          <p role="alert" className="text-destructive text-sm">
            {t("activity.invalidLimit")}
          </p>
        ) : null}
        {query.isError || save.isError ? (
          <p role="alert" className="text-destructive text-sm">
            {t(save.isError ? "activity.saveFailed" : "activity.failed")}
          </p>
        ) : null}
        {save.isSuccess ? (
          <p role="status" className="text-sm text-emerald-600 dark:text-emerald-400">
            {t("activity.saved")}
          </p>
        ) : null}
        <div className="space-y-3 border-t pt-4">
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0">
              <label htmlFor={contentToggleId} className="text-sm font-medium">
                {t("activity.contentSettingTitle")}
              </label>
              <p className="text-muted-foreground mt-1 text-xs leading-5">
                {t("activity.contentSettingDescription")}
              </p>
            </div>
            <Switch
              id={contentToggleId}
              checked={query.data?.contentEnabled ?? false}
              disabled={!query.isSuccess || contentToggle.isPending}
              onCheckedChange={(checked) => contentToggle.mutate(checked)}
            />
          </div>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="text-muted-foreground text-xs leading-5">
              {t("activity.contentRetentionHint")}
            </p>
            <Button
              variant="outline"
              size="sm"
              disabled={!query.isSuccess || contentClear.isPending}
              onClick={() => contentClear.mutate()}
            >
              <Trash2 aria-hidden="true" />
              {t(contentClear.isPending ? "activity.contentClearing" : "activity.contentClear")}
            </Button>
          </div>
          <p className="text-muted-foreground text-xs leading-5 tabular-nums">
            {t("activity.contentUsage", {
              used: formatBytes(query.data?.contentBytes ?? 0),
              budget: formatBytes(query.data?.contentBudget ?? 0),
            })}
          </p>
          {contentToggle.isError || contentClear.isError ? (
            <p role="alert" className="text-destructive text-sm">
              {t(
                contentToggle.isError
                  ? "activity.contentSettingFailed"
                  : "activity.contentClearFailed",
              )}
            </p>
          ) : null}
          {contentClear.data !== undefined ? (
            <p role="status" className="text-sm text-emerald-600 dark:text-emerald-400">
              {t("activity.contentCleared", { count: contentClear.data })}
            </p>
          ) : null}
        </div>
      </CardContent>
    </Card>
  )
}
