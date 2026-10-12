import type { ComponentProps } from "react"
import { Tabs as TabsPrimitive } from "radix-ui"
import { cn } from "../../lib/utils"

export const Tabs = TabsPrimitive.Root

export function TabsList({ className, ...props }: ComponentProps<typeof TabsPrimitive.List>) {
  return (
    <TabsPrimitive.List
      {...props}
      className={cn("flex min-w-0 flex-wrap gap-1 border-b", className)}
    />
  )
}

export function TabsTrigger({ className, ...props }: ComponentProps<typeof TabsPrimitive.Trigger>) {
  return (
    <TabsPrimitive.Trigger
      {...props}
      className={cn(
        "text-muted-foreground hover:bg-muted/50 hover:text-foreground data-[state=active]:border-primary data-[state=active]:text-foreground focus-visible:ring-ring/50 inline-flex min-h-11 cursor-pointer items-center justify-center gap-2 border-b-2 border-transparent px-4 py-2.5 text-sm font-medium transition-colors outline-none focus-visible:ring-2 disabled:pointer-events-none disabled:opacity-50 motion-reduce:transition-none [&_svg]:size-4 [&_svg]:shrink-0",
        className,
      )}
    />
  )
}

export function TabsContent({ className, ...props }: ComponentProps<typeof TabsPrimitive.Content>) {
  return (
    <TabsPrimitive.Content
      {...props}
      className={cn(
        "focus-visible:ring-ring/50 min-w-0 space-y-5 rounded-lg outline-none focus-visible:ring-2 data-[state=inactive]:hidden",
        className,
      )}
    />
  )
}
