import { useRef, useState, type FormEvent, type RefObject } from "react"
import { X } from "lucide-react"
import { Dialog } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Input } from "../../components/ui/input"
import { SelectField, type SelectOption } from "../../components/ui/select-field"
import { Switch } from "../../components/ui/switch"
import type {
  ProviderConfiguration,
  ProxyDiscovered,
  ProxySaveInput,
  ProxySettings,
  ProxyTestResult,
} from "../../services/omniproxy"
import { isKnownProviderKind, providerNameKeys } from "./provider-labels"
import { useProviders } from "./use-providers"
import {
  useDiscoverProxyPorts,
  useProxySettings,
  useSaveProxySettings,
  useSetProviderProxyMode,
  useTestProxySettings,
} from "./use-proxy-settings"

const SCHEMES = ["auto", "http", "socks5"] as const
type Scheme = (typeof SCHEMES)[number]

const MODES = ["inherit", "direct", "proxy"] as const
type ProxyMode = (typeof MODES)[number]

const MODE_LABEL_KEYS = {
  inherit: "providers.proxy.modeInherit",
  direct: "providers.proxy.modeDirect",
  proxy: "providers.proxy.modeProxy",
} as const

function isScheme(value: string): value is Scheme {
  return (SCHEMES as readonly string[]).includes(value)
}

function toMode(value: string | undefined): ProxyMode {
  return (MODES as readonly (string | undefined)[]).includes(value)
    ? (value as ProxyMode)
    : "inherit"
}

/** 协议名是协议本身的名字，不随界面语言变化。 */
function schemeLabel(scheme: Scheme): string {
  return scheme === "auto" ? "HTTP/SOCKS5" : scheme.toUpperCase()
}

/** 探测结果可能给出显式协议，也可能只有能力布尔位。 */
function resolveScheme(scheme: string, socks5: boolean, http: boolean): Scheme {
  if (isScheme(scheme) && scheme !== "auto") return scheme
  if (socks5 && !http) return "socks5"
  if (http && !socks5) return "http"
  return "auto"
}

/** endpoint 形如 127.0.0.1:10808 或 [::1]:7890，回填时只保留主机部分。 */
function hostFromEndpoint(endpoint: string, port: number): string {
  const suffix = `:${port}`
  const host = endpoint.endsWith(suffix) ? endpoint.slice(0, -suffix.length) : endpoint
  return host.replace(/^\[|\]$/g, "")
}

/** 后端错误信息可能带上用户输入，回显前先抹掉密码。 */
function safeErrorDetail(error: unknown, password: string): string {
  const message = error instanceof Error ? error.message : ""
  const scrubbed = password ? message.split(password).join("••••") : message
  return scrubbed.trim()
}

export function ProxySettingsDialog({
  onClose,
}: {
  /** 关闭由页面控制：弹窗只在打开时挂载，关闭即释放表单状态。 */
  onClose: () => void
}) {
  const { t } = useTranslation()
  const settings = useProxySettings()
  const hostRef = useRef<HTMLInputElement>(null)
  return (
    <Dialog.Root
      open
      onOpenChange={(next) => {
        if (!next) onClose()
      }}
    >
      <Dialog.Portal>
        <Dialog.Overlay className="motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 fixed inset-0 z-40 bg-black/50" />
        <Dialog.Content
          aria-modal="true"
          className="bg-card text-card-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 fixed top-1/2 left-1/2 z-50 max-h-[calc(100dvh-2rem)] w-[min(560px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border p-6 shadow-xl outline-none"
          onOpenAutoFocus={(event) => {
            event.preventDefault()
            hostRef.current?.focus()
          }}
        >
          {settings.isError ? (
            <div className="space-y-4">
              <Dialog.Title className="text-base font-semibold">
                {t("providers.proxy.title")}
              </Dialog.Title>
              <Dialog.Description className="text-muted-foreground text-sm">
                {t("providers.proxy.description")}
              </Dialog.Description>
              <p role="alert" className="text-destructive text-sm">
                {t("providers.proxy.loadFailed")}
              </p>
              <Dialog.Close asChild>
                <Button type="button" variant="outline" className="cursor-pointer">
                  {t("providers.proxy.cancel")}
                </Button>
              </Dialog.Close>
            </div>
          ) : settings.isPending ? (
            <div className="space-y-4">
              <Dialog.Title className="text-base font-semibold">
                {t("providers.proxy.title")}
              </Dialog.Title>
              <Dialog.Description className="text-muted-foreground text-sm">
                {t("providers.proxy.description")}
              </Dialog.Description>
              <p role="status" className="text-muted-foreground text-sm">
                {t("providers.loading")}
              </p>
            </div>
          ) : (
            <ProxySettingsForm
              key={settings.data.updatedAt}
              initial={settings.data}
              hostRef={hostRef}
              onClose={onClose}
            />
          )}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

function ProxySettingsForm({
  initial,
  hostRef,
  onClose,
}: {
  initial: ProxySettings
  hostRef: RefObject<HTMLInputElement | null>
  onClose: () => void
}) {
  const { t } = useTranslation()
  const providers = useProviders()
  const save = useSaveProxySettings()
  const test = useTestProxySettings()
  const detect = useDiscoverProxyPorts()
  const setMode = useSetProviderProxyMode()

  const [enabled, setEnabled] = useState(initial.enabled)
  const [scheme, setScheme] = useState<Scheme>(() =>
    isScheme(initial.scheme) ? initial.scheme : "auto",
  )
  const [host, setHost] = useState(initial.host)
  const [port, setPort] = useState(initial.port > 0 ? String(initial.port) : "")
  const [username, setUsername] = useState(initial.username)
  const [password, setPassword] = useState("")
  const [clearPassword, setClearPassword] = useState(false)
  const [noProxy, setNoProxy] = useState(initial.noProxy)
  const [detected, setDetected] = useState<ProxyDiscovered[] | null>(null)
  const [testResult, setTestResult] = useState<{
    result?: ProxyTestResult
    error?: boolean
  } | null>(null)
  const [saveFailed, setSaveFailed] = useState(false)
  const [saveDetail, setSaveDetail] = useState("")
  const [providerError, setProviderError] = useState(false)

  const portNumber = Number.parseInt(port, 10)
  const portValid = Number.isInteger(portNumber) && portNumber >= 1 && portNumber <= 65535
  const busy = save.isPending
  const hasSavedPassword = initial.hasPassword && !clearPassword

  const schemeOptions: SelectOption[] = [
    { value: "auto", label: t("providers.proxy.schemeAuto") },
    { value: "http", label: t("providers.proxy.schemeHttp") },
    { value: "socks5", label: t("providers.proxy.schemeSocks5") },
  ]

  const buildInput = (): ProxySaveInput => ({
    enabled,
    scheme,
    host: host.trim(),
    port: portValid ? portNumber : 0,
    username: username.trim(),
    password,
    clearPassword,
    noProxy: noProxy.trim(),
  })

  const providerName = (item: ProviderConfiguration) =>
    item.source === "builtin" && isKnownProviderKind(item.kind)
      ? t(providerNameKeys[item.kind])
      : item.name

  const builtins = providers.data?.filter((item) => item.source === "builtin") ?? []
  const custom = providers.data?.filter((item) => item.source !== "builtin") ?? []
  const listed = [...builtins, ...custom]

  const runDetect = async () => {
    setDetected(null)
    try {
      setDetected((await detect.mutateAsync()) ?? [])
    } catch {
      setDetected([])
    }
  }

  const applyDetected = (item: ProxyDiscovered) => {
    setHost(hostFromEndpoint(item.endpoint, item.port))
    setPort(String(item.port))
    setScheme(resolveScheme(item.scheme, item.socks5, item.http))
    setTestResult(null)
  }

  const runTest = async () => {
    setTestResult(null)
    try {
      setTestResult({ result: await test.mutateAsync(buildInput()) })
    } catch {
      setTestResult({ error: true })
    }
  }

  const changeMode = async (item: ProviderConfiguration, mode: ProxyMode) => {
    if (toMode(item.proxyMode) === mode) return
    setProviderError(false)
    try {
      await setMode.mutateAsync({ id: item.id, mode })
    } catch {
      setProviderError(true)
    }
  }

  const changeAll = async (mode: ProxyMode) => {
    setProviderError(false)
    try {
      for (const item of listed) {
        if (toMode(item.proxyMode) === mode) continue
        await setMode.mutateAsync({ id: item.id, mode })
      }
    } catch {
      setProviderError(true)
    }
  }

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!portValid) return
    setSaveFailed(false)
    setSaveDetail("")
    try {
      await save.mutateAsync(buildInput())
      onClose()
    } catch (error) {
      setSaveFailed(true)
      setSaveDetail(safeErrorDetail(error, password))
    }
  }

  const outcome = (): { tone: "ok" | "error"; text: string } | null => {
    if (!testResult) return null
    if (testResult.error || !testResult.result) {
      return { tone: "error", text: t("providers.testError") }
    }
    const result = testResult.result
    if (result.reachable) {
      return {
        tone: "ok",
        text: t("providers.proxy.testReachable", {
          scheme: schemeLabel(resolveScheme(result.scheme, result.socks5, result.http)),
          latency: result.latencyMs,
        }),
      }
    }
    return {
      tone: "error",
      text: t(
        result.errorCode === "proxy_protocol_mismatch"
          ? "providers.proxy.testMismatch"
          : "providers.proxy.testUnavailable",
      ),
    }
  }
  const testOutcome = outcome()

  return (
    <form onSubmit={submit} className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-1.5">
          <Dialog.Title className="text-base font-semibold">
            {t("providers.proxy.title")}
          </Dialog.Title>
          <Dialog.Description className="text-muted-foreground text-sm">
            {t("providers.proxy.description")}
          </Dialog.Description>
        </div>
        <Dialog.Close asChild>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            disabled={busy}
            aria-label={t("providers.proxy.cancel")}
            className="shrink-0 cursor-pointer"
          >
            <X className="size-4" aria-hidden="true" />
          </Button>
        </Dialog.Close>
      </div>

      <section aria-labelledby="proxy-server-title" className="space-y-4">
        <h3 id="proxy-server-title" className="text-sm font-semibold">
          {t("providers.proxy.server")}
        </h3>
        <div className="flex items-center justify-between gap-4">
          <label htmlFor="proxy-enabled" className="text-sm font-medium">
            {t("providers.proxy.enabled")}
          </label>
          <Switch
            id="proxy-enabled"
            aria-label={t("providers.proxy.enabled")}
            checked={enabled}
            disabled={busy}
            onCheckedChange={setEnabled}
          />
        </div>
        <div className="space-y-1.5">
          <label htmlFor="proxy-scheme" className="text-sm font-medium">
            {t("providers.proxy.scheme")}
          </label>
          <SelectField
            id="proxy-scheme"
            value={scheme}
            disabled={busy}
            options={schemeOptions}
            onValueChange={(value) => {
              if (isScheme(value)) setScheme(value)
            }}
          />
        </div>
        <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_7rem]">
          <div className="space-y-1.5">
            <label htmlFor="proxy-host" className="text-sm font-medium">
              {t("providers.proxy.host")}
            </label>
            <Input
              ref={hostRef}
              id="proxy-host"
              value={host}
              disabled={busy}
              maxLength={255}
              placeholder="127.0.0.1"
              onChange={(event) => setHost(event.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <label htmlFor="proxy-port" className="text-sm font-medium">
              {t("providers.proxy.port")}
            </label>
            <Input
              id="proxy-port"
              type="number"
              inputMode="numeric"
              min={1}
              max={65535}
              value={port}
              disabled={busy}
              aria-invalid={!portValid}
              placeholder="10808"
              onChange={(event) => setPort(event.target.value)}
            />
          </div>
        </div>
        {portValid || port.trim() === "" ? null : (
          <p role="alert" className="text-destructive text-xs">
            {t("providers.proxy.portInvalid")}
          </p>
        )}
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <label htmlFor="proxy-username" className="text-sm font-medium">
              {t("providers.proxy.username")}
            </label>
            <Input
              id="proxy-username"
              value={username}
              disabled={busy}
              autoComplete="off"
              maxLength={255}
              onChange={(event) => setUsername(event.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <label htmlFor="proxy-password" className="text-sm font-medium">
              {t("providers.proxy.password")}
            </label>
            <Input
              id="proxy-password"
              type="password"
              value={password}
              disabled={busy || clearPassword}
              autoComplete="new-password"
              maxLength={255}
              placeholder={hasSavedPassword ? t("providers.proxy.passwordKeep") : undefined}
              onChange={(event) => setPassword(event.target.value)}
            />
          </div>
        </div>
        <label className="flex cursor-pointer items-center gap-2 text-sm">
          <input
            type="checkbox"
            className="size-4"
            checked={clearPassword}
            disabled={busy || !initial.hasPassword}
            onChange={(event) => {
              setClearPassword(event.currentTarget.checked)
              if (event.currentTarget.checked) setPassword("")
            }}
          />
          {t("providers.proxy.clearPassword")}
        </label>
        <div className="space-y-1.5">
          <label htmlFor="proxy-no-proxy" className="text-sm font-medium">
            {t("providers.proxy.noProxy")}
          </label>
          <Input
            id="proxy-no-proxy"
            value={noProxy}
            disabled={busy}
            maxLength={2048}
            placeholder="localhost, 127.0.0.1, .internal"
            onChange={(event) => setNoProxy(event.target.value)}
          />
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => void runDetect()}
            disabled={detect.isPending || busy}
            className="cursor-pointer"
          >
            {detect.isPending ? t("providers.proxy.detecting") : t("providers.proxy.detect")}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => void runTest()}
            disabled={test.isPending || busy || !portValid}
            className="cursor-pointer"
          >
            {test.isPending ? t("providers.testing") : t("providers.proxy.test")}
          </Button>
        </div>
        {detected ? (
          detected.length > 0 ? (
            <ul className="flex flex-wrap gap-2">
              {detected.map((item) => (
                <li key={`${item.endpoint}-${item.scheme}`}>
                  <Button
                    type="button"
                    variant="secondary"
                    size="xs"
                    onClick={() => applyDetected(item)}
                    disabled={busy}
                    className="cursor-pointer font-mono"
                  >
                    {item.endpoint} ·{" "}
                    {schemeLabel(resolveScheme(item.scheme, item.socks5, item.http))}
                  </Button>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-muted-foreground text-xs leading-5">
              {t("providers.proxy.detectEmpty")}
            </p>
          )
        ) : null}
        {testOutcome ? (
          <p
            role={testOutcome.tone === "error" ? "alert" : "status"}
            className={`text-xs leading-5 ${testOutcome.tone === "error" ? "text-destructive" : "text-emerald-700 dark:text-emerald-300"}`}
          >
            {testOutcome.text}
          </p>
        ) : null}
      </section>

      <section aria-labelledby="proxy-providers-title" className="space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 id="proxy-providers-title" className="text-sm font-semibold">
            {t("providers.proxy.providers")}
          </h3>
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              variant="outline"
              size="xs"
              onClick={() => void changeAll("proxy")}
              disabled={busy || setMode.isPending || listed.length === 0}
              className="cursor-pointer"
            >
              {t("providers.proxy.allProxy")}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="xs"
              onClick={() => void changeAll("direct")}
              disabled={busy || setMode.isPending || listed.length === 0}
              className="cursor-pointer"
            >
              {t("providers.proxy.allDirect")}
            </Button>
          </div>
        </div>
        {providers.isError ? (
          <p role="alert" className="text-destructive text-sm">
            {t("providers.loadFailed")}
          </p>
        ) : null}
        {providerError ? (
          <p role="alert" className="text-destructive text-sm">
            {t("providers.actionFailed")}
          </p>
        ) : null}
        {listed.length > 0 ? (
          <ul className="divide-border divide-y rounded-lg border">
            {listed.map((item) => {
              const name = providerName(item)
              const current = toMode(item.proxyMode)
              return (
                <li
                  key={item.id}
                  className="flex flex-wrap items-center justify-between gap-2 px-3 py-2"
                >
                  <span className="min-w-0 truncate text-sm">{name}</span>
                  <div
                    role="group"
                    aria-label={t("providers.proxy.modesFor", { name })}
                    className="bg-muted flex flex-wrap gap-0.5 rounded-lg p-0.5"
                  >
                    {MODES.map((mode) => (
                      <Button
                        key={mode}
                        type="button"
                        variant="ghost"
                        size="xs"
                        aria-pressed={current === mode}
                        disabled={busy || setMode.isPending}
                        onClick={() => void changeMode(item, mode)}
                        className="aria-pressed:bg-background hover:bg-background/60 cursor-pointer aria-pressed:shadow-xs"
                      >
                        {t(MODE_LABEL_KEYS[mode])}
                      </Button>
                    ))}
                  </div>
                </li>
              )
            })}
          </ul>
        ) : null}
      </section>

      {saveFailed ? (
        <div role="alert" className="text-destructive space-y-1 text-sm">
          <p>{t("providers.proxy.saveFailed")}</p>
          {saveDetail ? <p className="text-xs break-words opacity-80">{saveDetail}</p> : null}
        </div>
      ) : null}

      <div className="flex flex-wrap gap-2">
        <Button type="submit" disabled={busy || !portValid} className="cursor-pointer">
          {busy ? t("providers.saving") : t("providers.proxy.save")}
        </Button>
        <Dialog.Close asChild>
          <Button type="button" variant="outline" disabled={busy} className="cursor-pointer">
            {t("providers.proxy.cancel")}
          </Button>
        </Dialog.Close>
      </div>
    </form>
  )
}
