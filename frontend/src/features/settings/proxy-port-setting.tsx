import type { FormEvent } from "react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card"
import { Input } from "../../components/ui/input"
import { useProxyPort, useSetProxyPort } from "./use-proxy-port"

const minPort = 1
const maxPort = 65535

function parsePort(value: string): number | null {
  const port = Number(value.trim())
  if (!Number.isInteger(port) || port < minPort || port > maxPort) {
    return null
  }
  return port
}

export function ProxyPortSetting() {
  const { t } = useTranslation()
  const state = useProxyPort()
  const save = useSetProxyPort()
  // 输入框本身不受控：后台刷新端口时用 key 重建，避免把正在编辑的内容覆盖掉。
  const [draft, setDraft] = useState("")
  const current = state.data
  const locked = current?.envLocked ?? false
  const invalid = draft.trim() !== "" && parsePort(draft) === null

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const port = parsePort(new FormData(event.currentTarget).get("port")?.toString() ?? "")
    if (port === null || locked) {
      return
    }
    save.mutate(port)
  }

  return (
    <Card className="gap-0 rounded-2xl py-4 shadow-none">
      <CardHeader className="px-5 pb-4">
        <CardTitle className="text-base">{t("settings.proxyPort.title")}</CardTitle>
        <CardDescription>{t("settings.proxyPort.description")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-3 px-5">
        {state.isError ? (
          <p className="text-destructive text-xs" role="alert">
            {t("settings.proxyPort.loadFailed", { error: state.error.message })}
          </p>
        ) : null}
        <div className="text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-xs">
          {current ? (
            <>
              <span>{t("settings.proxyPort.current", { port: current.port })}</span>
              <span>{t("settings.proxyPort.default", { port: current.defaultPort })}</span>
              <span>
                {current.available
                  ? t("settings.proxyPort.available")
                  : t("settings.proxyPort.unavailable")}
              </span>
              <span>
                {current.running
                  ? t("settings.proxyPort.running")
                  : t("settings.proxyPort.stopped")}
              </span>
            </>
          ) : (
            <span>{t(state.isPending ? "settings.proxyPort.loading" : "gateway.unavailable")}</span>
          )}
        </div>
        {locked ? (
          <p className="text-muted-foreground text-xs" role="status">
            {t("settings.proxyPort.envLocked")}
          </p>
        ) : null}
        {!locked && current ? (
          <form
            key={current.port}
            className="flex flex-wrap items-center gap-2"
            onSubmit={submit}
            noValidate
          >
            <Input
              id="proxy-port"
              name="port"
              className="w-32 font-mono text-xs"
              inputMode="numeric"
              aria-label={t("settings.proxyPort.label")}
              aria-invalid={invalid ? true : undefined}
              defaultValue={String(current.port)}
              disabled={save.isPending}
              onChange={(event) => setDraft(event.target.value)}
            />
            <Button type="submit" size="sm" disabled={save.isPending || invalid}>
              {save.isPending ? t("settings.proxyPort.saving") : t("settings.proxyPort.save")}
            </Button>
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={save.isPending}
              onClick={() => save.mutate(0)}
            >
              {t("settings.proxyPort.reset")}
            </Button>
          </form>
        ) : null}
        {invalid ? (
          <p className="text-destructive text-xs" role="alert">
            {t("settings.proxyPort.invalid")}
          </p>
        ) : null}
        {save.isError ? (
          <p className="text-destructive text-xs" role="alert">
            {t("settings.proxyPort.failed", { error: save.error.message })}
          </p>
        ) : null}
        {save.isSuccess && save.data.adjusted ? (
          <p className="text-xs" role="status">
            {t("settings.proxyPort.adjusted", {
              requested: save.data.requested,
              port: save.data.state.port,
            })}
          </p>
        ) : null}
        {save.isSuccess ? (
          <p className="text-muted-foreground text-xs" role="status">
            {t("settings.proxyPort.saved", { port: save.data.state.port })}
            {(save.data.updated ?? []).length > 0
              ? t("settings.proxyPort.rebased", { clients: (save.data.updated ?? []).join(" / ") })
              : ""}
          </p>
        ) : null}
        {save.isSuccess && (save.data.manual ?? []).length > 0 ? (
          <p className="text-destructive text-xs" role="alert">
            {t("settings.proxyPort.manual", { clients: (save.data.manual ?? []).join(" / ") })}
          </p>
        ) : null}
      </CardContent>
    </Card>
  )
}
