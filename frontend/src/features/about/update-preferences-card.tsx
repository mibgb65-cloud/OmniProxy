import { useTranslation } from "react-i18next"
import { Card } from "../../components/ui/card"
import { SelectField } from "../../components/ui/select-field"
import { Switch } from "../../components/ui/switch"
import type { useUpdateChannel } from "./use-update-channel"

export function UpdatePreferencesCard({
  settings,
  disabled,
}: {
  settings: ReturnType<typeof useUpdateChannel>
  disabled: boolean
}) {
  const { t } = useTranslation()
  const { preferences, save, automatic, download } = settings
  const unavailable =
    disabled ||
    preferences.isPending ||
    preferences.isError ||
    save.isPending ||
    automatic.isPending ||
    download.isPending
  return (
    <Card className="gap-0 overflow-hidden py-0 shadow-none">
      <div className="grid items-center gap-3 p-4 @min-[28rem]/updates:grid-cols-[minmax(0,1fr)_13rem] @min-[28rem]/updates:gap-6">
        <div className="min-w-0 space-y-1">
          <label htmlFor="update-channel" className="text-sm font-medium">
            {t("updates.channel")}
          </label>
          <p id="update-channel-hint" className="text-muted-foreground text-xs leading-relaxed">
            {t("updates.channelHint")}
          </p>
        </div>
        <SelectField
          id="update-channel"
          aria-describedby="update-channel-hint"
          value={preferences.data?.channel ?? "stable"}
          disabled={unavailable}
          onValueChange={(value) => save.mutate(value)}
          options={[
            { value: "stable", label: t("updates.stable") },
            { value: "beta", label: t("updates.beta") },
          ]}
        />
      </div>
      <div className="flex items-start justify-between gap-6 border-t p-4">
        <div className="min-w-0 space-y-1">
          <label
            htmlFor="automatic-update-check"
            className="flex flex-wrap gap-x-2 text-sm font-medium"
          >
            <span>{t("updates.automaticTitle")}</span>
          </label>
          <p id="automatic-update-hint" className="text-muted-foreground text-xs leading-relaxed">
            {t("updates.manual")}
          </p>
        </div>
        <Switch
          id="automatic-update-check"
          aria-label={t("appUpdate.autoCheck")}
          aria-describedby="automatic-update-hint"
          className="mt-0.5"
          checked={preferences.data?.autoCheck ?? true}
          disabled={unavailable}
          onCheckedChange={(enabled) => automatic.mutate(enabled)}
        />
      </div>
      <div className="flex items-start justify-between gap-6 border-t p-4">
        <div className="min-w-0 space-y-1">
          <label htmlFor="automatic-update-download" className="text-sm font-medium">
            {t("updates.autoDownload")}
          </label>
          <p id="automatic-download-hint" className="text-muted-foreground text-xs leading-relaxed">
            {t("updates.autoDownloadHint")}
          </p>
        </div>
        <Switch
          id="automatic-update-download"
          aria-describedby="automatic-download-hint"
          className="mt-0.5"
          checked={preferences.data?.autoDownload ?? false}
          disabled={unavailable}
          onCheckedChange={(enabled) => download.mutate(enabled)}
        />
      </div>
      {preferences.isError || save.isError || automatic.isError || download.isError ? (
        <p role="alert" className="text-destructive border-t px-4 py-3 text-sm">
          {t("updates.settingsFailed")}
        </p>
      ) : null}
    </Card>
  )
}
