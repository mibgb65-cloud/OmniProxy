import { useTranslation } from "react-i18next"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card"
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
    <Card className="dark:bg-card/75 gap-0">
      <CardHeader className="pb-5">
        <CardTitle className="text-base">{t("settings.desktop.title")}</CardTitle>
        <CardDescription>{t("settings.desktop.description")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5 text-sm">
        {isPending ? <p className="text-muted-foreground">{t("dashboard.loading")}</p> : null}
        {isError ? (
          <p className="text-destructive" role="alert">
            {t("settings.desktop.loadFailed", { error: error.message })}
          </p>
        ) : null}

        {data ? (
          <>
            <label
              className={`bg-muted/15 flex items-center justify-between gap-4 rounded-lg border px-4 py-3 ${
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
                  data.autoStart ? "bg-blue-600" : "bg-input"
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

            <fieldset
              className="space-y-3 border-t pt-5 disabled:opacity-60"
              disabled={closeBehavior.isPending}
            >
              <legend className="font-medium">{t("settings.desktop.closeTitle")}</legend>
              <p className="text-muted-foreground text-xs">
                {t("settings.desktop.closeDescription")}
              </p>
              <div className="grid gap-2 sm:grid-cols-2">
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
      className={`flex cursor-pointer items-start gap-3 rounded-lg border px-4 py-3 ${
        checked
          ? "border-blue-300 bg-blue-50 dark:border-blue-500/50 dark:bg-blue-500/5"
          : "bg-muted/15"
      }`}
    >
      <input
        type="radio"
        name="close-behavior"
        value={value}
        checked={checked}
        onChange={onChange}
        className="mt-0.5 accent-blue-600"
      />
      <span>
        <span className="block font-medium">{label}</span>
        <span className="text-muted-foreground mt-1 block text-xs">{description}</span>
      </span>
    </label>
  )
}
