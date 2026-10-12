import { SelectField } from "../../components/ui/select-field"
import { keepPreviousData, useQuery } from "@tanstack/react-query"
import { RefreshCw } from "lucide-react"
import { useDeferredValue, useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Input } from "../../components/ui/input"
import { readRequests } from "../../services/activity"
import { RequestTable } from "./request-table"
import { RequestDialog } from "./request-dialog"

export function RequestsView({
  requestId = "",
  onOpenLogs,
}: {
  requestId?: string
  onOpenLogs?: (id: string) => void
}) {
  const { t } = useTranslation()
  const [search, setSearch] = useState(requestId)
  const [state, setState] = useState("")
  const [offset, setOffset] = useState(0)
  const [selected, setSelected] = useState<string | null>(requestId || null)
  const deferred = useDeferredValue(search)
  const data = useQuery({
    queryKey: ["requests", deferred, state, offset],
    queryFn: () => readRequests({ search: deferred, state, offset, limit: 50 }),
    placeholderData: keepPreviousData,
    refetchInterval: 5000,
  })
  const items = data.data?.items ?? []
  return (
    <section className="w-full min-w-0 space-y-5 pb-6">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-semibold tracking-tight">{t("pages.requests.title")}</h1>
        <Button variant="outline" disabled={data.isFetching} onClick={() => void data.refetch()}>
          <RefreshCw
            className={data.isFetching ? "motion-safe:animate-spin" : ""}
            aria-hidden="true"
          />
          {t("activity.refresh")}
        </Button>
      </header>
      <div className="flex flex-wrap gap-3">
        <Input
          aria-label={t("activity.search")}
          placeholder={t("activity.search")}
          value={search}
          maxLength={128}
          className="min-w-0 flex-1"
          onChange={(e) => {
            setSearch(e.target.value)
            setOffset(0)
          }}
        />
        <SelectField
          aria-label={t("activity.state")}
          className="w-auto min-w-36"
          value={state}
          onValueChange={(value) => {
            setState(value)
            setOffset(0)
          }}
          options={(
            [
              ["", "activity.all"],
              ["success", "activity.success"],
              ["failed", "activity.failedState"],
              ["cancelled", "activity.cancelled"],
              ["pending", "activity.pending"],
            ] as const
          ).map(([value, label]) => ({ value, label: t(label) }))}
        />
      </div>
      {data.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("activity.failed")}
        </p>
      ) : null}
      {data.isPending ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("activity.loading")}
        </p>
      ) : items.length ? (
        <RequestTable items={items} onSelect={setSelected} />
      ) : (
        <p className="text-muted-foreground rounded-xl border p-8 text-center text-sm">
          {t("activity.noRequests")}
        </p>
      )}
      <footer className="text-muted-foreground flex flex-wrap items-center justify-between gap-3 text-xs">
        <span>
          {t("activity.retained", {
            count: data.data?.total ?? 0,
            limit: data.data?.maxRecords ?? 10000,
          })}
        </span>
        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            size="sm"
            disabled={offset === 0 || data.isFetching}
            onClick={() => setOffset(Math.max(0, offset - 50))}
          >
            {t("activity.previous")}
          </Button>
          <span>
            {t("activity.page", {
              page: Math.floor(offset / 50) + 1,
              count: data.data?.total ?? 0,
            })}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={offset + 50 >= (data.data?.total ?? 0) || data.isFetching}
            onClick={() => setOffset(offset + 50)}
          >
            {t("activity.next")}
          </Button>
        </div>
      </footer>
      <RequestDialog id={selected} onClose={() => setSelected(null)} onOpenLogs={onOpenLogs} />
    </section>
  )
}
