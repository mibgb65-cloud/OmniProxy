import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card"
import { GatewayStartupSetting } from "./gateway-startup-setting"
import {
  useDesktopSettings,
  useSetAutoStart,
  useSetCloseBehavior,
  type CloseBehavior,
} from "./use-desktop-settings"

const CLOSE_CHOICES = [
  {
    value: "tray",
    label: "settings.desktop.closeToTray",
    description: "settings.desktop.closeToTrayDescription",
  },
  {
    value: "exit",
    label: "settings.desktop.closeToExit",
    description: "settings.desktop.closeToExitDescription",
  },
] as const

export function DesktopBehavior() {
  const { t } = useTranslation()
  const { data, isPending, isError, error } = useDesktopSettings()
  const autoStart = useSetAutoStart()
  const closeBehavior = useSetCloseBehavior()

  return (
    <Card className="gap-0 rounded-2xl py-0 shadow-none">
      <CardHeader className="px-5 pt-4">
        <CardTitle className="text-muted-foreground text-xs font-medium">
          {t("settings.desktop.title")}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3 px-5 pb-4 text-sm">
        {isPending ? <p className="text-muted-foreground">{t("dashboard.loading")}</p> : null}
        {isError ? (
          <p className="text-destructive" role="alert">
            {t("settings.desktop.loadFailed", { error: error.message })}
          </p>
        ) : null}

        {data ? (
          <>
            <label
              className={`flex items-center justify-between gap-4 py-4 ${
                data.autoStartAvailable && !autoStart.isPending
                  ? "cursor-pointer"
                  : "cursor-default"
              }`}
            >
              <span className="min-w-0">
                <span className="block font-medium">{t("settings.desktop.autoStart")}</span>
                <span className="text-muted-foreground mt-1 block text-xs">
                  {data.autoStartAvailable
                    ? t("settings.desktop.autoStartDescription")
                    : t("settings.desktop.autoStartUnavailable")}
                </span>
              </span>
              <input
                type="checkbox"
                role="switch"
                className="peer sr-only"
                checked={data.autoStart}
                disabled={autoStart.isPending || !data.autoStartAvailable}
                onChange={(event) => autoStart.mutate(event.currentTarget.checked)}
              />
              <span
                className={`peer-focus-visible:ring-ring/50 relative h-6 w-11 shrink-0 rounded-full peer-focus-visible:ring-4 ${
                  data.autoStart ? "bg-primary" : "bg-input"
                }`}
                aria-hidden="true"
              >
                <span
                  className={`absolute top-0.5 left-0.5 size-5 rounded-full bg-white shadow-sm motion-safe:transition-transform ${
                    data.autoStart ? "translate-x-5" : "translate-x-0"
                  }`}
                />
              </span>
            </label>
            {autoStart.isError ? (
              <p className="text-destructive text-xs" role="alert">
                {t("settings.desktop.autoStartFailed", { error: autoStart.error.message })}
              </p>
            ) : null}

            <GatewayStartupSetting enabled={data.gatewayAutoStart} />
            <fieldset
              className="min-w-0 space-y-2 border-t pt-4 disabled:opacity-60"
              disabled={closeBehavior.isPending}
            >
              <legend className="sr-only">{t("settings.desktop.closeTitle")}</legend>
              <div className="flex flex-wrap items-center justify-between gap-3">
                <p className="font-medium" aria-hidden="true">
                  {t("settings.desktop.closeTitle")}
                </p>
                <div className="flex flex-wrap gap-2">
                  {CLOSE_CHOICES.map((choice) => (
                    <CloseChoice
                      key={choice.value}
                      value={choice.value}
                      label={t(choice.label)}
                      description={t(choice.description)}
                      checked={data.closeBehavior === choice.value}
                      onChange={() => closeBehavior.mutate(choice.value)}
                    />
                  ))}
                </div>
              </div>
              <p className="text-muted-foreground text-xs leading-5">
                {t(
                  data.closeBehavior === "tray"
                    ? "settings.desktop.closeToTrayDescription"
                    : "settings.desktop.closeToExitDescription",
                )}
              </p>
            </fieldset>
            {closeBehavior.isError ? (
              <p className="text-destructive text-xs" role="alert">
                {t("settings.desktop.closeFailed", { error: closeBehavior.error.message })}
              </p>
            ) : null}
            {autoStart.isPending || closeBehavior.isPending ? (
              <p className="text-muted-foreground text-xs" role="status">
                {t("settings.desktop.saving")}
              </p>
            ) : null}
          </>
        ) : null}
      </CardContent>
    </Card>
  )
}

function CloseChoice({
  value,
  label,
  description,
  checked,
  onChange,
}: {
  value: CloseBehavior
  label: string
  description: string
  checked: boolean
  onChange: () => void
}) {
  return (
    <label
      className={`flex cursor-pointer items-center gap-2 rounded-lg border px-3 py-2 ${
        checked ? "border-primary/40 bg-muted" : "hover:bg-muted/50"
      }`}
    >
      <input
        type="radio"
        name="close-behavior"
        value={value}
        checked={checked}
        onChange={onChange}
        className="accent-primary shrink-0"
      />
      <span>
        <span className="block text-xs font-medium">{label}</span>
        <span className="sr-only">{description}</span>
      </span>
    </label>
  )
}
