import type { ComponentProps } from "react"
import { Switch as SwitchPrimitive } from "radix-ui"
import { cn } from "../../lib/utils"

export function Switch({ className, ...props }: ComponentProps<typeof SwitchPrimitive.Root>) {
  return (
    <SwitchPrimitive.Root
      {...props}
      data-slot="switch"
      className={cn(
        "focus-visible:ring-ring/50 data-[state=unchecked]:bg-input inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent outline-none focus-visible:ring-4 disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-blue-600 dark:data-[state=checked]:bg-blue-400",
        className,
      )}
    >
      <SwitchPrimitive.Thumb className="pointer-events-none block size-4 translate-x-0 rounded-full bg-white shadow-sm data-[state=checked]:translate-x-4 motion-safe:transition-transform" />
    </SwitchPrimitive.Root>
  )
}
