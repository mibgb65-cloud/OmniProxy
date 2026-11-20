import { keepPreviousData, useQuery } from "@tanstack/react-query"
import { useDeferredValue, useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Input } from "../../components/ui/input"
import { readRecordedRequests, readRetention } from "../../services/activity"
import { RecordCard } from "./record-card"

const PAGE_SIZE = 20

export function RecordsView({ onOpenRequest }: { onOpenRequest: (id: string) => void }) {
  const { t } = useTranslation()
  const [search, setSearch] = useState("")
  const [offset, setOffset] = useState(0)
  const deferred = useDeferredValue(search)
  const retention = useQuery({ queryKey: ["request-retention"], queryFn: readRetention })
  const data = useQuery({
    queryKey: ["recorded-requests", deferred, offset],
    queryFn: () => readRecordedRequests({ search: deferred, offset, limit: PAGE_SIZE }),
    placeholderData: keepPreviousData,
    refetchInterval: 10000,
  })
  const items = data.data?.items ?? []
  const total = data.data?.total ?? 0
  const closed = retention.isSuccess && retention.data.contentEnabled === false
  const waiting = !closed && data.isSuccess && total === 0 && !deferred
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))
  return (
    <section className="app-page-fit flex w-full min-w-0 flex-col gap-4">
      <header className="shrink-0">
        <h1 className="text-2xl font-semibold tracking-tight">{t("records.title")}</h1>
        <p className="text-muted-foreground mt-1 text-sm">{t("records.description")}</p>
      </header>
      <Input
        aria-label={t("records.search")}
        placeholder={t("records.search")}
        value={search}
        maxLength={128}
        className="shrink-0"
        onChange={(event) => {
          setSearch(event.target.value)
          setOffset(0)
        }}
      />
      {closed ? (
        <div className="shrink-0 rounded-xl border border-amber-500/30 bg-amber-500/5 p-4 text-sm">
          <p className="font-medium">{t("records.offTitle")}</p>
          <p className="text-muted-foreground mt-1 text-xs leading-5">{t("records.offHint")}</p>
          {total > 0 ? (
            <p className="text-muted-foreground mt-2 text-xs">{t("records.offWithHistory")}</p>
          ) : null}
        </div>
      ) : null}
      {waiting ? (
        <div className="shrink-0 rounded-xl border p-4 text-sm">
          <p className="font-medium">{t("records.waitingTitle")}</p>
          <p className="text-muted-foreground mt-1 text-xs leading-5">{t("records.waitingHint")}</p>
        </div>
      ) : null}
      {data.isError ? (
        <p role="alert" className="text-destructive shrink-0 text-sm">
          {t("records.failed")}
        </p>
      ) : null}
      {data.isPending ? (
        <p role="status" className="text-muted-foreground shrink-0 text-sm">
          {t("records.loading")}
        </p>
      ) : items.length ? (
        <div className="app-scroll flex min-h-0 flex-1 flex-col gap-3 overflow-auto pr-1">
          {items.map((item) => (
            <RecordCard key={item.id} item={item} onOpen={onOpenRequest} />
          ))}
        </div>
      ) : (
        <p className="text-muted-foreground flex min-h-0 flex-1 items-center justify-center rounded-xl border p-8 text-center text-sm">
          {deferred ? t("records.noMatches") : t("records.empty")}
        </p>
      )}
      <footer className="text-muted-foreground flex shrink-0 flex-wrap items-center justify-between gap-3 text-xs">
        <span>{t("records.retained", { count: total })}</span>
        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            size="sm"
            disabled={offset === 0 || data.isFetching}
            onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
          >
            {t("records.previous")}
          </Button>
          <span>{t("records.page", { page: Math.floor(offset / PAGE_SIZE) + 1, totalPages })}</span>
          <Button
            variant="outline"
            size="sm"
            disabled={offset + PAGE_SIZE >= total || data.isFetching}
            onClick={() => setOffset(offset + PAGE_SIZE)}
          >
            {t("records.next")}
          </Button>
        </div>
      </footer>
    </section>
  )
}
