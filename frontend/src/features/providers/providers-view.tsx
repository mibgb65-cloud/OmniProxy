import { useState } from "react"
import { useTranslation } from "react-i18next"
import type { ProviderConfiguration, UpdateProviderInput } from "../../services/omniproxy"
import type { ProbeDisplay } from "./provider-card"
import { ProviderCatalog } from "./provider-catalog"
import { ProviderForm } from "./provider-form"
import { ProviderList } from "./provider-list"
import {
  useCreateProvider,
  useDeleteProvider,
  useProviderCatalog,
  useProviders,
  useProbeProvider,
  useSetProviderEnabled,
  useUpdateProvider,
} from "./use-providers"

type Editor = { initial?: ProviderConfiguration }

export function ProvidersView() {
  const { t } = useTranslation()
  const catalog = useProviderCatalog()
  const providers = useProviders()
  const create = useCreateProvider()
  const update = useUpdateProvider()
  const remove = useDeleteProvider()
  const setEnabled = useSetProviderEnabled()
  const probe = useProbeProvider()
  const [editor, setEditor] = useState<Editor | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [probeResults, setProbeResults] = useState<Record<string, ProbeDisplay>>({})
  const [probingId, setProbingId] = useState<string | null>(null)

  const builtins = providers.data?.filter((item) => item.source === "builtin") ?? []
  const custom = providers.data?.filter((item) => item.source === "custom") ?? []

  const submit = async (input: UpdateProviderInput) => {
    if (!editor) return
    if (editor.initial) {
      await update.mutateAsync({ id: editor.initial.id, input })
      const editedId = editor.initial.id
      setProbeResults((previous) => {
        const next = { ...previous }
        delete next[editedId]
        return next
      })
    } else {
      await create.mutateAsync(input)
    }
    setEditor(null)
  }

  const toggle = async (item: ProviderConfiguration, enabled: boolean) => {
    await setEnabled.mutateAsync({ id: item.id, enabled })
  }

  const toggleBuiltIn = (item: ProviderConfiguration, enabled: boolean) => {
    setActionError(null)
    void toggle(item, enabled).catch(() => setActionError(t("providers.actionFailed")))
  }

  const deleteItem = async (id: string) => {
    setActionError(null)
    try {
      await remove.mutateAsync(id)
      setProbeResults((previous) => {
        const next = { ...previous }
        delete next[id]
        return next
      })
      return true
    } catch (error) {
      const message = error instanceof Error ? error.message : ""
      setActionError(
        message.includes("provider: has accounts")
          ? t("providers.deleteHasAccounts")
          : t("providers.actionFailed"),
      )
      return false
    }
  }

  const testConnection = (item: ProviderConfiguration) => {
    if (probingId) return
    setProbingId(item.id)
    setProbeResults((previous) => ({ ...previous, [item.id]: {} }))
    void probe
      .mutateAsync(item.id)
      .then((result) => setProbeResults((previous) => ({ ...previous, [item.id]: { result } })))
      .catch(() => setProbeResults((previous) => ({ ...previous, [item.id]: { error: true } })))
      .finally(() => setProbingId(null))
  }

  return (
    <div className="mx-auto max-w-[1120px] space-y-7 pb-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t("providers.title")}</h1>
        <p className="text-muted-foreground mt-1 text-sm">{t("providers.description")}</p>
      </header>
      <p className="border-border bg-muted/30 rounded-lg border px-4 py-3 text-sm">
        {t("providers.configOnly")}
      </p>
      <p className="text-muted-foreground text-xs">{t("providers.previewHint")}</p>
      {catalog.isPending || providers.isPending ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("providers.loading")}
        </p>
      ) : null}
      {catalog.isError || providers.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("providers.loadFailed")}
        </p>
      ) : null}
      {catalog.data && providers.data ? (
        <ProviderCatalog
          definitions={catalog.data}
          items={builtins}
          busy={setEnabled.isPending}
          probingId={probingId}
          probeResults={probeResults}
          onToggle={toggleBuiltIn}
          onProbe={testConnection}
        />
      ) : null}
      {editor ? (
        <ProviderForm
          key={editor.initial?.id ?? "new-custom"}
          initial={editor.initial}
          onSubmit={submit}
          onCancel={() => setEditor(null)}
        />
      ) : null}
      {actionError ? (
        <p role="alert" className="text-destructive text-sm">
          {actionError}
        </p>
      ) : null}
      {providers.data ? (
        <ProviderList
          items={custom}
          definitions={catalog.data ?? []}
          onAdd={() => setEditor({})}
          onEdit={(item) => setEditor({ initial: item })}
          onToggle={toggle}
          onDelete={deleteItem}
          onProbe={testConnection}
          probeResults={probeResults}
          probingId={probingId}
          busy={setEnabled.isPending || update.isPending || remove.isPending}
        />
      ) : null}
    </div>
  )
}
