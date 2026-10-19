import { useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import type { UpdateStatus } from "../../services/omniproxy"
import { useCheckForUpdate, useRuntimeInfo } from "./use-about"
import { useUpdateChannel } from "./use-update-channel"
import { useApplicationUpdate } from "../app-update/use-application-update"
import { UpdateControls } from "../app-update/update-controls"
import { UpdateVersionCard } from "./update-version-card"
import { UpdatePreferencesCard } from "./update-preferences-card"
import { CurrentReleaseCard } from "./current-release-card"
import { UpdateCheckToast } from "./update-check-toast"

export function UpdateCard({ toastContainer }: { toastContainer?: HTMLElement | null }) {
  const { t } = useTranslation()
  const runtime = useRuntimeInfo()
  const info = runtime.data
  const check = useCheckForUpdate()
  const [notice, setNotice] = useState(0)
  const noticeSequence = useRef(0)
  const settings = useUpdateChannel(() => {
    check.reset()
    setNotice(0)
  })
  const { preferences, save, automatic, download } = settings
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
    <div className="@container/updates space-y-3">
      <UpdateVersionCard
        version={info?.version}
        checking={check.isPending || snapshot?.checkState === "checking"}
        disabled={
          check.isPending ||
          preferences.isPending ||
          preferences.isError ||
          save.isPending ||
          automatic.isPending ||
          download.isPending ||
          busy
        }
        onCheck={() => {
          setNotice(0)
          check.mutate(undefined, {
            onSuccess: (result) => {
              if (result.state === "up-to-date") setNotice(++noticeSequence.current)
            },
          })
        }}
      />
      <UpdatePreferencesCard settings={settings} disabled={check.isPending || busy} />
      {(status && ["available", "unconfigured", "error"].includes(status.state)) ||
      check.isError ||
      (snapshot && snapshot.download.state !== "idle") ? (
        <div className="space-y-3 text-sm" aria-live="polite">
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

          {status?.state === "unconfigured" ? (
            <p className="text-muted-foreground">{t("about.update.unconfigured")}</p>
          ) : null}

          {status?.state === "error" ? (
            <p className="text-destructive">
              {t("about.update.failed", { error: status.message })}
            </p>
          ) : null}

          {check.isError ? (
            <p className="text-destructive">
              {t("about.update.failed", { error: check.error.message })}
            </p>
          ) : null}
          {snapshot ? <UpdateControls state={snapshot} /> : null}
        </div>
      ) : null}
      {notice ? <UpdateCheckToast key={notice} container={toastContainer} /> : null}
      <CurrentReleaseCard
        version={info?.version}
        pending={runtime.isPending}
        failed={runtime.isError}
      />
    </div>
  )
}
