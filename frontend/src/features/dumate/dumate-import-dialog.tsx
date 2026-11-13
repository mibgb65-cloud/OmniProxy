import { useState } from "react"
import { Dialog } from "radix-ui"
import { LoaderCircle, MonitorSmartphone, ShieldCheck, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { useOpenGeneration } from "../../lib/use-open-generation"
import type { DuMateImportResult } from "../../services/omniproxy"
import { dumateErrorCode, dumateImportErrorKey } from "./dumate-error"
import { importSuccessKey, useDuMateImport } from "./use-dumate"

const MAX_COOKIE_BYTES = 64 << 10

/** 添加 DuMate 账号只有登录态导入，没有 API Key 或 OAuth 流程。 */
export function DuMateImportDialog({ providerId }: { providerId: string }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const generation = useOpenGeneration(open)
  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Trigger asChild>
        <Button type="button" variant="outline" size="sm" className="shrink-0 cursor-pointer">
          {t("dumate.add")}
        </Button>
      </Dialog.Trigger>
      {/* 常驻内容：退出动画需要关闭态有机会渲染；每次打开换 key 保证干净输入。 */}
      <ImportContent key={generation} providerId={providerId} />
    </Dialog.Root>
  )
}

function ImportContent({ providerId }: { providerId: string }) {
  const { t } = useTranslation()
  const [cookie, setCookie] = useState("")
  const [message, setMessage] = useState("")
  const [error, setError] = useState("")
  const imports = useDuMateImport(providerId)
  const busy = imports.desktop.isPending || imports.cookie.isPending
  const invalid = cookie.trim().length === 0 || cookie.length > MAX_COOKIE_BYTES

  const succeeded = (result: DuMateImportResult) => {
    setError("")
    setMessage(t(importSuccessKey(result), { name: result.account.name }))
  }
  const failed = (cause: unknown) => {
    setMessage("")
    setError(t(dumateImportErrorKey(dumateErrorCode(cause))))
  }
  const importDesktop = () => {
    setError("")
    setMessage("")
    imports.desktop.mutate(undefined, { onSuccess: succeeded, onError: failed })
  }
  const importCookie = () => {
    setError("")
    setMessage("")
    imports.cookie.mutate(cookie.trim(), { onSuccess: succeeded, onError: failed })
  }

  return (
    <Dialog.Portal>
      <Dialog.Overlay className="motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 fixed inset-0 z-40 bg-black/55 motion-safe:duration-150" />
      <Dialog.Content className="bg-card text-card-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 motion-safe:data-[state=closed]:zoom-out-95 fixed top-1/2 left-1/2 z-50 flex max-h-[calc(100dvh-2rem)] w-[min(560px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-2xl border shadow-2xl outline-none motion-safe:duration-150">
        <div className="flex shrink-0 items-start justify-between gap-4 px-6 pt-6 pb-5">
          <div className="min-w-0 space-y-1.5">
            <Dialog.Title className="text-lg font-semibold tracking-tight">
              {t("dumate.import.title")}
            </Dialog.Title>
            <Dialog.Description className="text-muted-foreground text-sm leading-6">
              {t("dumate.import.description")}
            </Dialog.Description>
          </div>
          <Dialog.Close asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              className="text-muted-foreground -mt-1 -mr-1 shrink-0 cursor-pointer"
              disabled={busy}
              aria-label={t("dumate.import.close")}
            >
              <X className="size-4" aria-hidden="true" />
            </Button>
          </Dialog.Close>
        </div>
        <div className="min-h-0 space-y-5 overflow-y-auto px-6 pb-6">
          <section className="space-y-2 rounded-xl border p-4">
            <h3 className="text-sm font-medium">{t("dumate.import.desktop")}</h3>
            <p className="text-muted-foreground text-xs leading-5">
              {t("dumate.import.desktopHint")}
            </p>
            <Button
              type="button"
              size="sm"
              disabled={busy}
              onClick={importDesktop}
              className="cursor-pointer"
            >
              {imports.desktop.isPending ? (
                <LoaderCircle
                  className="size-4 animate-spin motion-reduce:animate-none"
                  aria-hidden="true"
                />
              ) : (
                <MonitorSmartphone className="size-4" aria-hidden="true" />
              )}
              {t(imports.desktop.isPending ? "dumate.import.importing" : "dumate.import.desktop")}
            </Button>
          </section>
          <form
            className="space-y-2 rounded-xl border p-4"
            onSubmit={(event) => {
              event.preventDefault()
              if (!invalid && !busy) importCookie()
            }}
          >
            <label htmlFor="dumate-cookie" className="text-sm font-medium">
              {t("dumate.import.cookie")}
            </label>
            <p id="dumate-cookie-hint" className="text-muted-foreground text-xs leading-5">
              {t("dumate.import.cookieHint")}
            </p>
            <textarea
              id="dumate-cookie"
              rows={4}
              value={cookie}
              maxLength={MAX_COOKIE_BYTES}
              disabled={busy}
              onChange={(event) => setCookie(event.target.value)}
              autoComplete="off"
              autoCapitalize="none"
              spellCheck={false}
              placeholder={t("dumate.import.cookiePlaceholder")}
              aria-describedby="dumate-cookie-hint"
              className="border-input dark:bg-input/30 placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 min-h-24 w-full resize-y rounded-md border bg-transparent px-3 py-2 font-mono text-xs shadow-xs outline-none focus-visible:ring-[3px] disabled:opacity-50"
            />
            <Button type="submit" size="sm" disabled={busy || invalid} className="cursor-pointer">
              {imports.cookie.isPending ? (
                <LoaderCircle
                  className="size-4 animate-spin motion-reduce:animate-none"
                  aria-hidden="true"
                />
              ) : null}
              {t(imports.cookie.isPending ? "dumate.import.importing" : "dumate.import.submit")}
            </Button>
          </form>
          {message ? (
            <p
              role="status"
              aria-live="polite"
              className="text-sm text-emerald-600 dark:text-emerald-400"
            >
              {message}
            </p>
          ) : null}
          {error ? (
            <p role="alert" className="text-destructive text-sm">
              {error}
            </p>
          ) : null}
          <p className="text-muted-foreground flex items-start gap-2 text-xs leading-5">
            <ShieldCheck className="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
            {t("dumate.import.storage")}
          </p>
        </div>
        <div className="bg-muted/20 flex shrink-0 justify-end gap-2 border-t px-6 py-4">
          <Dialog.Close asChild>
            <Button
              type="button"
              variant="outline"
              disabled={busy}
              className="min-w-20 cursor-pointer"
            >
              {t("dumate.import.close")}
            </Button>
          </Dialog.Close>
        </div>
      </Dialog.Content>
    </Dialog.Portal>
  )
}
