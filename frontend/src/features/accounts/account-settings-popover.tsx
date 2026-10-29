import { useId, type ReactNode } from "react"
import { Popover } from "radix-ui"
import { Settings2, X } from "lucide-react"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import { ScrollRegion } from "../../components/common/scroll-region"

export function AccountSettingsPopover({
  title,
  closeLabel,
  children,
}: {
  title: string
  closeLabel: string
  children: ReactNode
}) {
  const titleId = useId()
  return (
    <Popover.Root modal={false}>
      <Tooltip content={title}>
        <Popover.Trigger asChild>
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            aria-label={title}
            className="cursor-pointer"
          >
            <Settings2 className="size-4" aria-hidden="true" />
          </Button>
        </Popover.Trigger>
      </Tooltip>
      <Popover.Portal>
        <Popover.Content
          side="bottom"
          align="start"
          sideOffset={8}
          collisionPadding={12}
          sticky="always"
          aria-labelledby={titleId}
          aria-describedby={undefined}
          className="bg-popover text-popover-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 motion-safe:data-[state=closed]:zoom-out-95 z-50 flex max-h-[min(560px,var(--radix-popover-content-available-height))] w-[min(420px,calc(100vw-1.5rem))] flex-col overflow-hidden rounded-xl border p-4 shadow-xl outline-none motion-safe:duration-150"
        >
          <div className="flex shrink-0 items-center justify-between gap-3 border-b pb-3">
            <h2 id={titleId} className="text-base font-semibold">
              {title}
            </h2>
            <Popover.Close asChild>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                aria-label={closeLabel}
                className="cursor-pointer"
              >
                <X className="size-3.5" aria-hidden="true" />
              </Button>
            </Popover.Close>
          </div>
          <ScrollRegion
            as="section"
            ariaLabel={title}
            className="min-h-0 flex-1"
            viewportClassName="pr-1"
          >
            {children}
          </ScrollRegion>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
