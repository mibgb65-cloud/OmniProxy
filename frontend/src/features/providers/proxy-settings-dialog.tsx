import { useRef, useState, type FormEvent, type ReactNode, type RefObject } from "react"
import { Globe, Search, X } from "lucide-react"
import { Dialog } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Input } from "../../components/ui/input"
import { SelectField, type SelectOption } from "../../components/ui/select-field"
import { Switch } from "../../components/ui/switch"
import { cn } from "../../lib/utils"
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

/** 字段标签统一用弱化的 12px 字重，避免每个输入框上方都是强对比大标签。 */
const LABEL_CLASS = "text-muted-foreground text-xs font-medium"

/**
 * 弹窗内的分区卡片：标题行（可带右侧操作）+ 内容。
 * 两栏布局靠它把「代理服务器」与「提供商分流」分开，而不是靠纵向堆叠的 h3。
 */
function SectionCard({
  titleId,
  title,
  aside,
  children,
}: {
  titleId: string
  title: string
  aside?: ReactNode
  children: ReactNode
}) {
  return (
    <section
      aria-labelledby={titleId}
      className="border-border/70 bg-muted/15 flex min-h-0 min-w-0 flex-col gap-3.5 rounded-xl border p-4"
    >
      <div className="flex min-h-6 flex-wrap items-center justify-between gap-2">
        <h3 id={titleId} className="text-sm font-semibold tracking-tight">
          {title}
        </h3>
        {aside}
      </div>
      {children}
    </section>
  )
}

/** 三态控件：整行右对齐成一列，未选中项弱化，选中项用背景抬起。 */
function ModeToggle({
  current,
  label,
  disabled,
  onChange,
}: {
  current: ProxyMode
  label: string
  disabled: boolean
  onChange: (mode: ProxyMode) => void
}) {
  const { t } = useTranslation()
  return (
    <div
      role="group"
      aria-label={label}
      className="bg-muted/70 inline-flex shrink-0 items-center gap-0.5 rounded-lg p-0.5"
    >
      {MODES.map((mode) => (
        <Button
          key={mode}
          type="button"
          variant="ghost"
          size="xs"
          aria-pressed={current === mode}
          disabled={disabled}
          onClick={() => onChange(mode)}
          className="text-muted-foreground hover:text-foreground aria-pressed:bg-background aria-pressed:text-foreground hover:bg-background/60 aria-pressed:hover:bg-background cursor-pointer rounded-[6px] text-xs font-medium aria-pressed:shadow-xs"
        >
          {t(MODE_LABEL_KEYS[mode])}
        </Button>
      ))}
    </div>
  )
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
          className="bg-card text-card-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 fixed top-1/2 left-1/2 z-50 flex max-h-[calc(100dvh-2rem)] w-[min(880px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-2xl border shadow-2xl outline-none"
          onOpenAutoFocus={(event) => {
            event.preventDefault()
            hostRef.current?.focus()
          }}
        >
          {settings.isError ? (
            <div className="space-y-4 p-6">
              <Dialog.Title className="text-lg font-semibold tracking-tight">
                {t("providers.proxy.title")}
              </Dialog.Title>
              <Dialog.Description className="text-muted-foreground text-sm leading-6">
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
            <div className="space-y-4 p-6">
              <Dialog.Title className="text-lg font-semibold tracking-tight">
                {t("providers.proxy.title")}
              </Dialog.Title>
              <Dialog.Description className="text-muted-foreground text-sm leading-6">
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
  const [filter, setFilter] = useState("")
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

  const query = filter.trim().toLowerCase()
  const visible = query
    ? listed.filter(
        (item) =>
          providerName(item).toLowerCase().includes(query) ||
          item.name.toLowerCase().includes(query),
      )
    : listed

  const counts = listed.reduce(
    (accumulated, item) => {
      accumulated[toMode(item.proxyMode)] += 1
      return accumulated
    },
    { inherit: 0, direct: 0, proxy: 0 },
  )

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
    <form onSubmit={submit} className="flex min-h-0 flex-1 flex-col">
      <div className="flex shrink-0 items-start justify-between gap-4 px-6 pt-6 pb-5">
        <div className="flex min-w-0 items-start gap-3">
          <div className="border-border/60 bg-muted/40 mt-0.5 hidden size-9 shrink-0 items-center justify-center rounded-lg border sm:flex">
            <Globe className="text-foreground/70 size-4.5" aria-hidden="true" />
          </div>
          <div className="min-w-0 space-y-1">
            <Dialog.Title className="text-lg font-semibold tracking-tight">
              {t("providers.proxy.title")}
            </Dialog.Title>
            <Dialog.Description className="text-muted-foreground text-sm leading-6">
              {t("providers.proxy.description")}
            </Dialog.Description>
          </div>
        </div>
        <Dialog.Close asChild>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            disabled={busy}
            aria-label={t("providers.proxy.cancel")}
            className="text-muted-foreground -mt-1 -mr-1 shrink-0 cursor-pointer"
          >
            <X className="size-4" aria-hidden="true" />
          </Button>
        </Dialog.Close>
      </div>

      <div className="app-scroll min-h-0 flex-1 overflow-y-auto px-6 pb-6">
        <div className="grid min-w-0 gap-4 md:grid-cols-2">
          <SectionCard titleId="proxy-server-title" title={t("providers.proxy.server")}>
            <div className="flex items-center justify-between gap-3">
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
            {enabled ? null : (
              <p className="text-muted-foreground -mt-1 text-xs leading-5">
                {t("providers.proxy.disabledHint")}
              </p>
            )}
            <div className="border-border/60 border-t" aria-hidden="true" />
            <div className="space-y-1.5">
              <label htmlFor="proxy-scheme" className={LABEL_CLASS}>
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
            <div className="grid grid-cols-[minmax(0,1fr)_6.5rem] gap-3">
              <div className="min-w-0 space-y-1.5">
                <label htmlFor="proxy-host" className={LABEL_CLASS}>
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
              <div className="min-w-0 space-y-1.5">
                <label htmlFor="proxy-port" className={LABEL_CLASS}>
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
              <p role="alert" className="text-destructive -mt-1 text-xs">
                {t("providers.proxy.portInvalid")}
              </p>
            )}
            <div className="space-y-1.5">
              <label htmlFor="proxy-username" className={LABEL_CLASS}>
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
              <div className="flex items-center justify-between gap-2">
                <label htmlFor="proxy-password" className={LABEL_CLASS}>
                  {t("providers.proxy.password")}
                </label>
                <label
                  className={cn(
                    "flex items-center gap-1.5 text-xs",
                    initial.hasPassword
                      ? "text-muted-foreground cursor-pointer"
                      : "text-muted-foreground/60 cursor-not-allowed",
                  )}
                >
                  <input
                    type="checkbox"
                    className="size-3.5"
                    checked={clearPassword}
                    disabled={busy || !initial.hasPassword}
                    onChange={(event) => {
                      setClearPassword(event.currentTarget.checked)
                      if (event.currentTarget.checked) setPassword("")
                    }}
                  />
                  {t("providers.proxy.clearPassword")}
                </label>
              </div>
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
            <div className="space-y-1.5">
              <label htmlFor="proxy-no-proxy" className={LABEL_CLASS}>
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
            <div className="border-border/60 border-t" aria-hidden="true" />
            <div className="space-y-2">
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
                  <ul className="flex flex-wrap gap-1.5">
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
            </div>
          </SectionCard>

          <SectionCard
            titleId="proxy-providers-title"
            title={t("providers.proxy.providers")}
            aside={
              <div className="flex shrink-0 items-center gap-1.5">
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
            }
          >
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
            {providers.isPending ? (
              <p className="text-muted-foreground text-xs leading-5">{t("providers.loading")}</p>
            ) : null}
            {listed.length > 0 ? (
              <>
                <div className="relative">
                  <Search
                    className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2"
                    aria-hidden="true"
                  />
                  <Input
                    type="search"
                    value={filter}
                    disabled={busy}
                    placeholder={t("providers.proxy.filterPlaceholder")}
                    aria-label={t("providers.proxy.filterPlaceholder")}
                    className="h-8 pl-8 text-xs md:text-xs"
                    onChange={(event) => setFilter(event.target.value)}
                  />
                </div>
                <ul className="border-border/70 divide-border/60 app-scroll bg-background/40 max-h-[min(22.5rem,50vh)] min-h-0 divide-y overflow-y-auto rounded-lg border">
                  {visible.map((item) => {
                    const name = providerName(item)
                    return (
                      <li
                        key={item.id}
                        className="hover:bg-accent/40 flex items-center justify-between gap-3 px-2 py-1.5"
                      >
                        <span className="min-w-0 truncate text-sm">{name}</span>
                        <ModeToggle
                          current={toMode(item.proxyMode)}
                          label={t("providers.proxy.modesFor", { name })}
                          disabled={busy || setMode.isPending}
                          onChange={(mode) => void changeMode(item, mode)}
                        />
                      </li>
                    )
                  })}
                  {visible.length === 0 ? (
                    <li className="text-muted-foreground px-3 py-6 text-center text-xs">
                      {t("providers.proxy.filterEmpty")}
                    </li>
                  ) : null}
                </ul>
                <p className="text-muted-foreground text-xs leading-5">
                  {query
                    ? t("providers.proxy.filterSummary", {
                        shown: visible.length,
                        total: listed.length,
                      })
                    : t("providers.proxy.summary", {
                        total: listed.length,
                        proxy: counts.proxy,
                        direct: counts.direct,
                        inherit: counts.inherit,
                      })}
                </p>
              </>
            ) : null}
          </SectionCard>
        </div>
      </div>

      <div className="bg-muted/20 flex shrink-0 flex-wrap items-center justify-end gap-3 border-t px-6 py-4">
        {saveFailed ? (
          <div role="alert" className="text-destructive mr-auto min-w-0 flex-1 space-y-0.5 text-xs">
            <p>{t("providers.proxy.saveFailed")}</p>
            {saveDetail ? <p className="truncate opacity-80">{saveDetail}</p> : null}
          </div>
        ) : null}
        <Dialog.Close asChild>
          <Button
            type="button"
            variant="outline"
            disabled={busy}
            className="min-w-20 cursor-pointer"
          >
            {t("providers.proxy.cancel")}
          </Button>
        </Dialog.Close>
        <Button type="submit" disabled={busy || !portValid} className="min-w-24 cursor-pointer">
          {busy ? t("providers.saving") : t("providers.proxy.save")}
        </Button>
      </div>
    </form>
  )
}
