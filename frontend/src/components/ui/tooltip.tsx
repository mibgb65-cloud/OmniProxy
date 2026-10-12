import { createContext, useContext, type ReactElement, type ReactNode } from "react"
import { Tooltip as TooltipPrimitive } from "radix-ui"

const ProviderContext = createContext(false)

export function TooltipProvider({ children }: { children: ReactNode }) {
  return (
    <ProviderContext value={true}>
      <TooltipPrimitive.Provider delayDuration={350} skipDelayDuration={150}>
        {children}
      </TooltipPrimitive.Provider>
    </ProviderContext>
  )
}

export function Tooltip({
  children,
  content,
  side = "top",
}: {
  children: ReactElement
  content?: ReactNode
  side?: "top" | "right" | "bottom" | "left"
}) {
  const provided = useContext(ProviderContext)
  if (!content) return children
  const tooltip = (
    <TooltipPrimitive.Root>
      <TooltipPrimitive.Trigger asChild>{children}</TooltipPrimitive.Trigger>
      <TooltipPrimitive.Portal>
        <TooltipPrimitive.Content
          side={side}
          sideOffset={8}
          collisionPadding={12}
          className="border-border/80 bg-popover/95 text-popover-foreground data-[state=delayed-open]:animate-in data-[state=delayed-open]:fade-in-0 data-[state=delayed-open]:zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[side=top]:slide-in-from-bottom-1 data-[side=bottom]:slide-in-from-top-1 z-[100] max-w-[min(24rem,calc(100vw-1.5rem))] rounded-xl border px-3 py-2 text-xs leading-relaxed break-words shadow-[0_8px_30px_-8px_rgb(0_0_0/0.25)] backdrop-blur-md motion-reduce:animate-none"
        >
          {content}
          <TooltipPrimitive.Arrow className="fill-popover" width={10} height={5} />
        </TooltipPrimitive.Content>
      </TooltipPrimitive.Portal>
    </TooltipPrimitive.Root>
  )
  // 独立页面和测试也可直接使用；应用中的提示共享延时，连续查看时无需反复等待。
  return provided ? tooltip : <TooltipProvider>{tooltip}</TooltipProvider>
}
