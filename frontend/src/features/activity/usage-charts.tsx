import type { ReactNode } from "react"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { cn } from "../../lib/utils"

export function UsageCard({
  title,
  hint,
  action,
  children,
  className,
  contentClassName,
}: {
  title: string
  hint?: ReactNode
  action?: ReactNode
  children: ReactNode
  className?: string
  contentClassName?: string
}) {
  return (
    <Card role="region" aria-label={title} className={cn("min-w-0 gap-3 py-4", className)}>
      <CardHeader className="gap-1 px-4">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div className="min-w-0">
            <h2 className="text-base font-semibold">{title}</h2>
            {hint ? <p className="text-muted-foreground mt-1 text-xs leading-5">{hint}</p> : null}
          </div>
          {action}
        </div>
      </CardHeader>
      <CardContent className={cn("min-w-0 px-4", contentClassName)}>{children}</CardContent>
    </Card>
  )
}

export function UsageEmpty({ children }: { children: ReactNode }) {
  return (
    <p className="text-muted-foreground flex min-h-24 items-center justify-center px-4 text-center text-sm">
      {children}
    </p>
  )
}

export function ChartLegend({
  items,
}: {
  items: ReadonlyArray<{ key: string; label: string; color: string; value?: ReactNode }>
}) {
  return (
    <div className="text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-xs">
      {items.map((item) => (
        <span key={item.key} className="inline-flex min-w-0 items-center gap-1.5">
          <span className={cn("size-2 shrink-0 rounded-full", item.color)} aria-hidden="true" />
          <span className="truncate">{item.label}</span>
          {item.value !== undefined ? (
            <span className="text-foreground/70 font-mono tabular-nums">{item.value}</span>
          ) : null}
        </span>
      ))}
    </div>
  )
}

export type DonutSegment = { key: string; value: bigint; className: string }

/**
 * 轻量 SVG 环形图：只按比例画弧，中心内容由调用方提供。
 * 比例使用 BigInt 计算，Token 超过 Number 安全范围也不会出现偏差。
 */
export function Donut({
  segments,
  label,
  size = 132,
  thickness = 13,
  center,
}: {
  segments: ReadonlyArray<DonutSegment>
  label: string
  size?: number
  thickness?: number
  center?: ReactNode
}) {
  const total = segments.reduce((sum, segment) => sum + segment.value, 0n)
  const radius = (size - thickness) / 2
  const circumference = 2 * Math.PI * radius
  const ratios = segments.map((segment) =>
    total > 0n ? Number((segment.value * 10000n) / total) / 10000 : 0,
  )
  const offsets = ratios.map((_, index) =>
    ratios.slice(0, index).reduce((sum, ratio) => sum + ratio, 0),
  )
  return (
    <div className="relative shrink-0" style={{ width: size, height: size }}>
      <svg
        viewBox={`0 0 ${size} ${size}`}
        className="size-full -rotate-90"
        role="img"
        aria-label={label}
      >
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          strokeWidth={thickness}
          className="stroke-muted"
        />
        {segments.map((segment, index) => {
          const ratio = ratios[index]!
          const dash = ratio * circumference
          return (
            <circle
              key={segment.key}
              cx={size / 2}
              cy={size / 2}
              r={radius}
              fill="none"
              strokeWidth={thickness}
              strokeDasharray={`${dash} ${circumference - dash}`}
              strokeDashoffset={-(offsets[index]! * circumference)}
              className={segment.className}
            />
          )
        })}
      </svg>
      {center ? (
        <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center text-center">
          {center}
        </div>
      ) : null}
    </div>
  )
}

/** 水平占比条；极小的非零值也保留 1px 可见宽度。 */
export function ShareBar({
  percent,
  className,
  segments,
}: {
  percent: number
  className?: string
  segments?: ReadonlyArray<{ key: string; percent: number; className: string }>
}) {
  const width = Math.min(100, Math.max(percent > 0 ? 1 : 0, percent))
  return (
    <div className="bg-muted h-2 overflow-hidden rounded-full">
      <div className="flex h-full overflow-hidden rounded-full" style={{ width: `${width}%` }}>
        {segments ? (
          segments.map((segment) => (
            <span
              key={segment.key}
              className={cn("h-full", segment.className)}
              style={{ width: `${Math.min(100, Math.max(0, segment.percent))}%` }}
            />
          ))
        ) : (
          <span className={cn("h-full w-full", className ?? "bg-blue-500")} />
        )}
      </div>
    </div>
  )
}
