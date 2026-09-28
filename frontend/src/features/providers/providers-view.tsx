import { useState } from "react"
import { useTranslation } from "react-i18next"
import type {
  ProviderConfiguration,
  ProviderDefinition,
  UpdateProviderInput,
} from "../../services/omniproxy"
import { ProviderCatalog } from "./provider-catalog"
import { ProviderForm } from "./provider-form"
import { ProviderList } from "./provider-list"
import {
  useCreateProvider,
  useDeleteProvider,
  useProviderCatalog,
  useProviders,
  useUpdateProvider,
} from "./use-providers"

type Editor = { kind: string; providerName: string; initial?: ProviderConfiguration }

export function ProvidersView() {
  const { t } = useTranslation()
  const catalog = useProviderCatalog()
  const providers = useProviders()
  const create = useCreateProvider()
  const update = useUpdateProvider()
  const remove = useDeleteProvider()
  const [editor, setEditor] = useState<Editor | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const items = providers.data ?? []
  const counts = items.reduce<Record<string, number>>((result, item) => {
    result[item.kind] = (result[item.kind] ?? 0) + 1
    return result
  }, {})

  const submit = async (input: UpdateProviderInput) => {
    if (!editor) return
    if (editor.initial) {
      await update.mutateAsync({ id: editor.initial.id, input })
    } else {
      await create.mutateAsync({ kind: editor.kind, ...input })
    }
    setEditor(null)
  }

  const toggle = async (item: ProviderConfiguration) => {
    await update.mutateAsync({
      id: item.id,
      input: { name: item.name, baseUrl: item.baseUrl, enabled: !item.enabled },
    })
  }

  const deleteItem = async (id: string) => {
    setDeleteError(null)
    try {
      await remove.mutateAsync(id)
      return true
    } catch (error) {
      const message = error instanceof Error ? error.message : ""
      setDeleteError(
        message.includes("provider: has accounts")
          ? t("providers.deleteHasAccounts")
          : t("providers.actionFailed"),
      )
      return false
    }
  }

  const openNew = (definition: ProviderDefinition, providerName: string) => {
    setEditor({ kind: definition.kind, providerName })
  }

  const openEdit = (item: ProviderConfiguration) => {
    const definition = catalog.data?.find((entry) => entry.kind === item.kind)
    setEditor({ kind: item.kind, providerName: definition?.name ?? item.kind, initial: item })
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
      {catalog.data ? (
        <ProviderCatalog definitions={catalog.data} counts={counts} onAdd={openNew} />
      ) : null}
      {editor ? (
        <ProviderForm
          key={editor.initial?.id ?? editor.kind}
          providerName={editor.providerName}
          initial={editor.initial}
          onSubmit={submit}
          onCancel={() => setEditor(null)}
        />
      ) : null}
      {deleteError ? (
        <p role="alert" className="text-destructive text-sm">
          {deleteError}
        </p>
      ) : null}
      {providers.data ? (
        <ProviderList
          items={items}
          onEdit={openEdit}
          onToggle={toggle}
          onDelete={deleteItem}
          busy={update.isPending || remove.isPending}
        />
      ) : null}
    </div>
  )
}
