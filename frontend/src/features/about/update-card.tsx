import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card"
import type { UpdateStatus } from "../../services/omniproxy"
import { useCheckForUpdate, useRuntimeInfo } from "./use-about"

export function UpdateCard() {
  const { t } = useTranslation()
  const { data: info } = useRuntimeInfo()
  const check = useCheckForUpdate()

  const status: UpdateStatus | undefined = check.data

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("about.update.title")}</CardTitle>
        <CardDescription>{t("about.update.description")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-3 text-sm">
        <div className="flex flex-wrap items-center gap-3">
          <span className="text-muted-foreground">
            {t("about.update.currentVersion", { version: info?.version ?? "…" })}
          </span>
          <Button
            size="sm"
            variant="outline"
            disabled={check.isPending}
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
      </CardContent>
    </Card>
  )
}
