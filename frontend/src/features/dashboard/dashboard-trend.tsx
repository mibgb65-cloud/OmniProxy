import { useCallback, useLayoutEffect, useRef, useState } from "react"
import type { PointerEvent, ReactNode } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { Tooltip } from "../../components/ui/tooltip"
import { formatTokens } from "../../services/activity"
import type { DashboardInfo, DashboardSummary } from "../../services/dashboard"
import {
  dashboardAxisTicks,
  dashboardLabelStep,
  dashboardPointLabel,
  dashboardTokenRatio,
  type DashboardPeriod,
} from "./dashboard-model"
import "./dashboard-trend.css"

type TrendMode = "requests" | "tokens"

/** 自绘滑块的最短长度，保证极长的横轴仍能抓住滑块。 */
const MIN_THUMB_WIDTH = 28
/** 每列最小宽度（px）：低于此宽度柱形与横轴标签会挤在一起。 */
const MIN_COLUMN_WIDTH = 22

function series(summary: DashboardSummary, mode: TrendMode) {
  return mode === "tokens"
    ? ([
        { key: "activity.input", value: summary.inputTokens, color: "bg-blue-500" },
        { key: "activity.output", value: summary.outputTokens, color: "bg-emerald-500" },
      ] as const)
    : ([
        { key: "activity.success", value: String(summary.success), color: "bg-emerald-500" },
        { key: "activity.failedState", value: String(summary.failed), color: "bg-rose-500" },
        { key: "activity.cancelled", value: String(summary.cancelled), color: "bg-zinc-400" },
        { key: "activity.pending", value: String(summary.pending), color: "bg-blue-500" },
      ] as const)
}

function compactFormatter(language: string) {
  return new Intl.NumberFormat(language === "zh" ? "zh-CN" : "en-US", {
    notation: "compact",
    maximumFractionDigits: 1,
  })
}

/**
 * 横向滚动容器 + 自绘滑块。
 * 原生滚动条在深色卡片里会带灰色轨道与两端箭头，这里隐藏原生滚动条，
 * 只在需要滚动时于图区下方显示 8px 圆角滑块，支持拖动与点击轨道跳转。
 */
function TrendScroller({ label, children }: { label: string; children: ReactNode }) {
  const viewportRef = useRef<HTMLDivElement>(null)
  const contentRef = useRef<HTMLDivElement>(null)
  const railRef = useRef<HTMLDivElement>(null)
  const thumbRef = useRef<HTMLDivElement>(null)
  const dragRef = useRef<{ pointerId: number; offset: number } | null>(null)

  const syncThumb = useCallback(() => {
    const viewport = viewportRef.current
    const rail = railRef.current
    const thumb = thumbRef.current
    if (!viewport || !rail || !thumb) return

    const maxScroll = viewport.scrollWidth - viewport.clientWidth
    const railWidth = rail.clientWidth
    const scrollable = maxScroll > 1 && railWidth > 0
    rail.dataset.scrollable = String(scrollable)
    if (!scrollable) return

    const thumbWidth = Math.min(
      railWidth,
      Math.max(MIN_THUMB_WIDTH, (railWidth * viewport.clientWidth) / viewport.scrollWidth),
    )
    const thumbLeft = (viewport.scrollLeft / maxScroll) * (railWidth - thumbWidth)
    thumb.style.width = `${thumbWidth}px`
    thumb.style.transform = `translateX(${thumbLeft}px)`
  }, [])

  useLayoutEffect(() => {
    const viewport = viewportRef.current
    const content = contentRef.current
    const rail = railRef.current
    if (!viewport || !content || !rail) return

    const observer = new ResizeObserver(syncThumb)
    observer.observe(viewport)
    observer.observe(content)
    observer.observe(rail)
    syncThumb()
    return () => observer.disconnect()
  }, [syncThumb])

  const scrollToPointer = (clientX: number, offset: number) => {
    const viewport = viewportRef.current
    const rail = railRef.current
    const thumb = thumbRef.current
    if (!viewport || !rail || !thumb) return

    const maxScroll = viewport.scrollWidth - viewport.clientWidth
    const travel = rail.clientWidth - thumb.offsetWidth
    if (maxScroll <= 0 || travel <= 0) return

    const position = clientX - rail.getBoundingClientRect().left - offset
    viewport.scrollLeft = (Math.max(0, Math.min(position, travel)) / travel) * maxScroll
    syncThumb()
  }

  const handlePointerDown = (event: PointerEvent<HTMLDivElement>) => {
    const rail = railRef.current
    const thumb = thumbRef.current
    if (event.button !== 0 || !rail || !thumb || rail.dataset.scrollable !== "true") return

    event.preventDefault()
    // 点击轨道时让滑块中心跳到指针位置；拖动滑块时保留原有抓取点。
    const offset =
      event.target === thumb
        ? event.clientX - thumb.getBoundingClientRect().left
        : thumb.offsetWidth / 2
    dragRef.current = { pointerId: event.pointerId, offset }
    rail.dataset.dragging = "true"
    rail.setPointerCapture(event.pointerId)
    scrollToPointer(event.clientX, offset)
  }

  const handlePointerMove = (event: PointerEvent<HTMLDivElement>) => {
    if (dragRef.current?.pointerId === event.pointerId) {
      scrollToPointer(event.clientX, dragRef.current.offset)
    }
  }

  const handlePointerEnd = (event: PointerEvent<HTMLDivElement>) => {
    const rail = railRef.current
    if (dragRef.current?.pointerId !== event.pointerId || !rail) return
    dragRef.current = null
    delete rail.dataset.dragging
    if (rail.hasPointerCapture(event.pointerId)) rail.releasePointerCapture(event.pointerId)
  }

  return (
    <div className="min-w-0 flex-1">
      <div
        ref={viewportRef}
        role="group"
        aria-label={label}
        className="trend-scroll"
        onScroll={syncThumb}
      >
        <div ref={contentRef}>{children}</div>
      </div>
      <div
        ref={railRef}
        aria-hidden="true"
        className="trend-rail"
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerEnd}
        onPointerCancel={handlePointerEnd}
        onLostPointerCapture={handlePointerEnd}
      >
        <div ref={thumbRef} className="trend-rail__thumb" />
      </div>
    </div>
  )
}

export function DashboardTrend({
  data,
  period,
  loading,
}: {
  data?: DashboardInfo
  period: DashboardPeriod
  loading: boolean
}) {
  const { t, i18n } = useTranslation()
  const [mode, setMode] = useState<TrendMode>("requests")
  const language = i18n.resolvedLanguage ?? "zh"
  const points = data?.points ?? []
  const columns = points.map((point) => {
    const segments = series(point.summary, mode)
    return {
      point,
      segments,
      total: segments.reduce((sum, item) => sum + BigInt(item.value), 0n),
    }
  })
  const maximum = columns.reduce((max, column) => (column.total > max ? column.total : max), 0n)
  const ticks = dashboardAxisTicks(maximum)
  const legend = data ? series(data.summary, mode) : []
  const compact = compactFormatter(language)
  const labelStep = dashboardLabelStep(columns.length)
  return (
    <Card className="min-w-0 gap-3 py-4">
      <CardHeader className="space-y-3 px-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-base font-semibold">{t("dash.trend")}</h2>
          <div
            role="group"
            aria-label={t("dash.trendMode")}
            className="bg-muted flex rounded-lg p-0.5"
          >
            {(["requests", "tokens"] as const).map((value) => (
              <Button
                key={value}
                variant={mode === value ? "secondary" : "ghost"}
                size="sm"
                aria-pressed={mode === value}
                className={
                  mode === value
                    ? "bg-background text-foreground shadow-xs"
                    : "text-muted-foreground"
                }
                onClick={() => setMode(value)}
              >
                {t(value === "requests" ? "dash.trendRequests" : "dash.trendTokens")}
              </Button>
            ))}
          </div>
        </div>
        <div className="text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-xs">
          {legend.map((item) => (
            <span key={item.key} className="inline-flex items-center gap-1.5">
              <span className={`size-2 rounded-full ${item.color}`} aria-hidden="true" />
              {t(item.key)}
              <span className="text-foreground/70 tabular-nums">
                {compact.format(BigInt(item.value))}
              </span>
            </span>
          ))}
        </div>
      </CardHeader>
      <CardContent className="min-w-0 px-4">
        {data && maximum > 0n ? (
          <div className="flex min-w-0 gap-2 pt-1">
            <div aria-hidden="true" className="relative h-52 w-9 shrink-0 sm:w-10">
              {ticks.map((value, index) => (
                <span
                  key={value.toString()}
                  className="text-muted-foreground absolute right-0 text-[10px] tabular-nums"
                  style={{
                    top: `${(index * 100) / 4}%`,
                    transform:
                      index === 0
                        ? undefined
                        : index === 4
                          ? "translateY(-100%)"
                          : "translateY(-50%)",
                  }}
                >
                  {compact.format(value)}
                </span>
              ))}
            </div>
            <TrendScroller label={t("dash.trend")}>
              <div
                className="relative"
                style={{ minWidth: Math.max(260, columns.length * MIN_COLUMN_WIDTH) }}
              >
                <div
                  className="pointer-events-none absolute inset-x-0 top-0 h-52"
                  aria-hidden="true"
                >
                  {ticks.slice(1, -1).map((value, index) => (
                    <div
                      key={value.toString()}
                      className="border-border/70 absolute inset-x-0 border-t border-dashed"
                      style={{ top: `${((index + 1) * 100) / 4}%` }}
                    />
                  ))}
                  <div className="border-border absolute inset-x-0 border-t" />
                  <div className="border-border absolute inset-x-0 bottom-0 border-t" />
                </div>
                <div
                  className="relative grid h-52 items-end"
                  style={{
                    gridTemplateColumns: `repeat(${columns.length}, minmax(0,1fr))`,
                  }}
                >
                  {columns.map(({ point, segments, total }) => {
                    const title = dashboardPointLabel(point.start, period, language, true)
                    return (
                      <Tooltip
                        key={point.start}
                        content={
                          <div className="min-w-40 space-y-1.5">
                            <p className="font-medium">{title}</p>
                            {segments.map((item) => (
                              <p key={item.key} className="flex items-center justify-between gap-6">
                                <span className="inline-flex items-center gap-1.5">
                                  <span
                                    className={`size-1.5 rounded-full ${item.color}`}
                                    aria-hidden="true"
                                  />
                                  {t(item.key)}
                                </span>
                                <span className="tabular-nums">
                                  {formatTokens(item.value, language)}
                                </span>
                              </p>
                            ))}
                            {segments.length > 1 ? (
                              <p className="flex items-center justify-between gap-6 border-t pt-1.5 font-medium">
                                <span>{t("dash.trendTotal")}</span>
                                <span className="tabular-nums">
                                  {formatTokens(total.toString(), language)}
                                </span>
                              </p>
                            ) : null}
                          </div>
                        }
                      >
                        <div
                          tabIndex={0}
                          aria-label={`${title} · ${t("dash.requests")} ${point.summary.requests} · ${t("dash.tokens")} ${point.summary.totalTokens}`}
                          className="group focus-visible:ring-ring/50 relative flex h-full min-w-0 flex-col justify-end rounded-sm outline-none focus-visible:ring-2"
                        >
                          <span
                            aria-hidden="true"
                            className="group-hover:bg-foreground/5 group-focus-visible:bg-foreground/5 absolute inset-x-0.5 inset-y-0 rounded-sm transition-colors"
                          />
                          <div className="relative flex h-full w-full items-end justify-center">
                            {total > 0n ? (
                              <div
                                className="flex w-[58%] max-w-[22px] min-w-[4px] flex-col-reverse overflow-hidden rounded-t-[3px]"
                                style={{
                                  height: `${dashboardTokenRatio(total.toString(), ticks[0]!)}%`,
                                  minHeight: 2,
                                }}
                              >
                                {segments.map((item) => (
                                  <div
                                    key={item.key}
                                    className={item.color}
                                    style={{
                                      height: `${dashboardTokenRatio(item.value, total)}%`,
                                      minHeight: BigInt(item.value) > 0n ? 1 : 0,
                                    }}
                                  />
                                ))}
                              </div>
                            ) : null}
                          </div>
                        </div>
                      </Tooltip>
                    )
                  })}
                </div>
                <div
                  className="text-muted-foreground mt-2 grid text-[10px] tabular-nums"
                  style={{
                    gridTemplateColumns: `repeat(${columns.length}, minmax(0,1fr))`,
                  }}
                >
                  {columns.map(({ point }, index) => (
                    <span
                      key={point.start}
                      className={
                        index === 0
                          ? "text-start whitespace-nowrap"
                          : index === columns.length - 1
                            ? "text-end whitespace-nowrap"
                            : "text-center whitespace-nowrap"
                      }
                    >
                      {index % labelStep === 0 || index === columns.length - 1
                        ? period === 1
                          ? String(new Date(point.start * 1000).getHours()).padStart(2, "0")
                          : dashboardPointLabel(point.start, period, language)
                        : ""}
                    </span>
                  ))}
                </div>
              </div>
            </TrendScroller>
          </div>
        ) : (
          <p
            role={loading ? "status" : undefined}
            className="text-muted-foreground flex min-h-52 items-center justify-center px-4 text-center text-sm"
          >
            {t(
              loading
                ? "dash.loading"
                : data
                  ? mode === "tokens" &&
                    data.summary.requests > 0 &&
                    data.summary.completeUsage === 0
                    ? "dash.trendUsageUnknown"
                    : "dash.trendEmpty"
                  : "dash.trendUnavailable",
            )}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
