import { useId } from "react"
import { useTranslation } from "react-i18next"
import { Switch } from "../../components/ui/switch"
import { useSetGatewayAutoStart } from "./use-desktop-settings"

export function GatewayStartupSetting({ enabled }: { enabled: boolean }) {
  const { t } = useTranslation()
  const toggleId = useId()
  const setting = useSetGatewayAutoStart()

  return (
    <div className="border-t pt-4">
      <label
        htmlFor={toggleId}
        className={`flex items-center justify-between gap-4 ${
          setting.isPending ? "cursor-default opacity-60" : "cursor-pointer"
        }`}
      >
        <span className="min-w-0">
          <span className="block font-medium">{t("settings.desktop.gatewayAutoStart")}</span>
          <span className="text-muted-foreground mt-1 block text-xs">
            {t("settings.desktop.gatewayAutoStartDescription")}
          </span>
        </span>
        <Switch
          id={toggleId}
          checked={enabled}
          disabled={setting.isPending}
          onCheckedChange={(checked) => setting.mutate(checked)}
        />
      </label>
      {setting.isPending ? (
        <p className="text-muted-foreground mt-2 text-xs" role="status">
          {t("settings.desktop.saving")}
        </p>
      ) : null}
      {setting.isError ? (
        <p className="text-destructive mt-2 text-xs" role="alert">
          {t("settings.desktop.gatewayAutoStartFailed", { error: setting.error.message })}
        </p>
      ) : null}
    </div>
  )
}
