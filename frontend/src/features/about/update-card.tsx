import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card"
import type { UpdateStatus } from "../../services/omniproxy"
import { useCheckForUpdate, useRuntimeInfo } from "./use-about"
import { SelectField } from "../../components/ui/select-field"
import { useUpdateChannel } from "./use-update-channel"
import { useApplicationUpdate } from "../app-update/use-application-update"
import { UpdateControls } from "../app-update/update-controls"

export function UpdateCard() {
  const { t } = useTranslation()
  const { data: info } = useRuntimeInfo()
  const check = useCheckForUpdate()
  const { preferences, save, automatic } = useUpdateChannel(() => check.reset())
  const application = useApplicationUpdate()

  const snapshot = application.data
  const busy =
    ["downloading", "verifying", "installing"].includes(snapshot?.download.state ?? "") ||
    snapshot?.checkState === "checking"
  const status: UpdateStatus | undefined =
    snapshot && snapshot.checkState !== "idle"
      ? {
          state: snapshot.checkState,
          currentVersion: snapshot.currentVersion,
          latestVersion: snapshot.release?.version ?? "",
          releaseUrl: snapshot.release?.url ?? "",
          publishedAt: snapshot.release?.publishedAt ?? "",
          message: snapshot.checkError,
        }
      : check.data

  return (
    <Card className="gap-4 rounded-2xl py-4 shadow-none">
      <CardHeader className="px-5">
        <CardTitle>{t("about.update.title")}</CardTitle>
        <CardDescription>{t("about.update.description")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-3 px-5 text-sm">
        <div className="space-y-2">
          <label htmlFor="update-channel" className="font-medium">
            {t("updates.channel")}
          </label>
          <SelectField
            id="update-channel"
            className="max-w-xs"
            value={preferences.data?.channel ?? "stable"}
            disabled={
              preferences.isPending ||
              preferences.isError ||
              save.isPending ||
              check.isPending ||
              busy
            }
            onValueChange={(value) => save.mutate(value)}
            options={[
              { value: "stable", label: t("updates.stable") },
              { value: "beta", label: t("updates.beta") },
            ]}
          />
          <p className="text-muted-foreground">{t("updates.manual")}</p>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={preferences.data?.autoCheck ?? true}
              disabled={preferences.isPending || automatic.isPending}
              onChange={(event) => automatic.mutate(event.target.checked)}
            />
            {t("appUpdate.autoCheck")}
          </label>
          {preferences.isError || save.isError ? (
            <p role="alert" className="text-destructive">
              {t("updates.settingsFailed")}
            </p>
          ) : null}
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <span className="text-muted-foreground">
            {t("about.update.currentVersion", { version: info?.version ?? "…" })}
          </span>
          <Button
            size="sm"
            variant="outline"
            disabled={
              check.isPending ||
              preferences.isPending ||
              preferences.isError ||
              save.isPending ||
              busy
            }
            onClick={() => check.mutate()}
          >
            {check.isPending ? t("about.update.checking") : t("about.update.check")}
          </Button>
        </div>

        {status?.state === "available" ? (
          <div className="bg-muted rounded-md px-3 py-2">
            <p className="font-medium">{t("about.update.available")}</p>
            <p className="text-muted-foreground mt-1">
              {t("about.update.latestVersion", { version: status.latestVersion })}
            </p>
            {status.releaseUrl ? (
              <a
                className="text-primary mt-1 inline-block underline underline-offset-4"
                href={status.releaseUrl}
                target="_blank"
                rel="noreferrer"
              >
                {t("about.update.openRelease")}
              </a>
            ) : null}
          </div>
        ) : null}

        {status?.state === "up-to-date" ? (
          <p className="text-muted-foreground">{t("about.update.upToDate")}</p>
        ) : null}

        {status?.state === "unconfigured" ? (
          <p className="text-muted-foreground">{t("about.update.unconfigured")}</p>
        ) : null}

        {status?.state === "error" ? (
          <p className="text-destructive">{t("about.update.failed", { error: status.message })}</p>
        ) : null}

        {check.isError ? (
          <p className="text-destructive">
            {t("about.update.failed", { error: check.error.message })}
          </p>
        ) : null}
        {snapshot ? <UpdateControls state={snapshot} /> : null}
      </CardContent>
    </Card>
  )
}
