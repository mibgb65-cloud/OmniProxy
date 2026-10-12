import { Dialog } from "radix-ui"
import { useLayoutEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { Tabs, TabsContent } from "../../components/ui/tabs"
import { RoutingHeader } from "./routing-header"
import type { RouteInput, RouteItem } from "../../services/routing"
import { RoutingDeleteDialog } from "./routing-delete-dialog"
import { RoutingEditor } from "./routing-editor"
import { RoutingPreview } from "./routing-preview"
import { RoutingRules } from "./routing-rules"
import { isRoutingTab, routingTabs, type RoutingTab } from "./routing-tabs"
import { RoutingTabList } from "./routing-tab-list"
import { useRouting } from "./use-routing"
import { ModelCatalog } from "./model-catalog"
import { DefaultModelRoutes } from "./default-model-routes"

export function RoutingView({
  tab,
  onTabChange,
}: {
  tab: RoutingTab
  onTabChange: (tab: RoutingTab) => void
}) {
  const { t } = useTranslation()
  const data = useRouting()
  const pageRef = useRef<HTMLDivElement>(null)
  const [visited, setVisited] = useState<RoutingTab[]>([tab])
  // 首次访问后保留面板，切换与历史导航不丢失输入、同步状态和异步结果。
  if (!visited.includes(tab)) setVisited([...visited, tab])
  useLayoutEffect(() => {
    const viewport = pageRef.current?.closest<HTMLElement>(".scroll-region__viewport")
    if (viewport) viewport.scrollTop = 0
  }, [tab])
  const returnFocus = useRef<HTMLElement | null>(null)
  const addRef = useRef<HTMLButtonElement | null>(null)
  const rememberFocus = () => {
    returnFocus.current =
      document.activeElement instanceof HTMLElement ? document.activeElement : null
  }
  const restoreFocus = () => {
    if (returnFocus.current?.isConnected && !returnFocus.current.closest("[hidden]")) {
      returnFocus.current.focus()
      return
    }
    addRef.current?.focus()
  }
  const [open, setOpen] = useState(false)
  const [editor, setEditor] = useState<RouteItem | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<RouteItem | null>(null)
  const [actionError, setActionError] = useState(false)
  const ready =
    data.routes.isSuccess &&
    data.providers.isSuccess &&
    data.accounts.isSuccess &&
    data.gateway.isSuccess
  const pending =
    data.routes.isPending ||
    data.providers.isPending ||
    data.accounts.isPending ||
    data.gateway.isPending
  const failed =
    data.routes.isError || data.providers.isError || data.accounts.isError || data.gateway.isError
  const busy =
    data.create.isPending ||
    data.update.isPending ||
    data.remove.isPending ||
    data.toggle.isPending ||
    data.preference.isPending
  const models = [
    ...new Set([
      ...(data.catalog.data ?? []).map((m) => m.model),
      ...(data.gateway.data?.models?.filter((id) => id.startsWith("deepseek-")) ?? []),
    ]),
  ]
  async function save(input: RouteInput) {
    if (editor) await data.update.mutateAsync({ id: editor.rule.id, input })
    else await data.create.mutateAsync(input)
    data.preview.reset()
    setOpen(false)
  }
  async function remove() {
    if (!deleteTarget) return
    setActionError(false)
    try {
      await data.remove.mutateAsync(deleteTarget.rule.id)
      data.preview.reset()
      setDeleteTarget(null)
    } catch {
      setActionError(true)
    }
  }
  function toggle(id: string, enabled: boolean) {
    setActionError(false)
    data.preview.reset()
    data.toggle.mutate({ id, enabled }, { onError: () => setActionError(true) })
  }
  const panels = {
    defaults: <DefaultModelRoutes data={data} />,
    rules: ready ? (
      <RoutingRules
        items={data.routes.data ?? []}
        busy={busy}
        onEdit={(item) => {
          rememberFocus()
          setEditor(item)
          setOpen(true)
        }}
        onDelete={(item) => {
          rememberFocus()
          setActionError(false)
          setDeleteTarget(item)
        }}
        onToggle={toggle}
      />
    ) : null,
    catalog: (
      <ModelCatalog accounts={data.accounts.data ?? []} providers={data.providers.data ?? []} />
    ),
    preview: <RoutingPreview data={data} />,
  }
  return (
    <Tabs
      ref={pageRef}
      value={tab}
      activationMode="manual"
      onValueChange={(value) => {
        if (isRoutingTab(value)) onTabChange(value)
      }}
      className="w-full min-w-0 space-y-5 pb-6"
    >
      <div className="bg-background/95 sticky top-0 z-20 space-y-5 pt-1 backdrop-blur-sm">
        <RoutingHeader
          fetching={data.isFetching}
          disabled={!ready || busy}
          addRef={addRef}
          onRefresh={() => void data.refresh()}
          onAdd={() => {
            rememberFocus()
            onTabChange("rules")
            setEditor(null)
            setOpen(true)
          }}
        />
        <RoutingTabList />
      </div>
      {pending ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("routing.loading")}
        </p>
      ) : null}
      {failed ? (
        <p role="alert" className="text-destructive text-sm">
          {t("routing.loadFailed")}
        </p>
      ) : null}
      {actionError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("routing.actionFailed")}
        </p>
      ) : null}
      {routingTabs.map((value) => (
        <TabsContent
          key={value}
          value={value}
          forceMount={visited.includes(value) ? true : undefined}
          hidden={tab !== value}
        >
          {panels[value]}
        </TabsContent>
      ))}
      <Dialog.Root
        open={open}
        onOpenChange={(next) => {
          if (!busy) setOpen(next)
        }}
      >
        {open && ready ? (
          <RoutingEditor
            key={editor?.rule.id ?? "new"}
            initial={editor}
            providers={data.providers.data ?? []}
            accounts={data.accounts.data ?? []}
            models={models}
            catalog={data.catalog.data ?? []}
            onSubmit={save}
            onClose={() => setOpen(false)}
            onRestoreFocus={restoreFocus}
          />
        ) : null}
      </Dialog.Root>
      <RoutingDeleteDialog
        item={deleteTarget}
        busy={data.remove.isPending}
        onCancel={() => setDeleteTarget(null)}
        onConfirm={() => void remove()}
        onRestoreFocus={restoreFocus}
        error={actionError}
      />
    </Tabs>
  )
}
