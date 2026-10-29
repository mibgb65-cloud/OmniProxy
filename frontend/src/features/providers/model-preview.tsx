import { useId } from "react"
import { ChevronDown, X } from "lucide-react"
import { Popover } from "radix-ui"
import { useTranslation } from "react-i18next"
import { ScrollRegion } from "../../components/common/scroll-region"
import { Button } from "../../components/ui/button"

export function ModelPreview({ name, models }: { name: string; models: string[] }) {
  const { t } = useTranslation()
  const titleId = useId()
  const title = `${name} · ${t("providers.modelsTitle")}`
  if (models.length === 0) {
    return <p className="text-muted-foreground text-xs">{t("providers.customModels")}</p>
  }
  return (
    <Popover.Root modal={false}>
      <Popover.Trigger asChild>
        <button
          type="button"
          className="group text-muted-foreground hover:text-foreground focus-visible:ring-ring flex max-w-full cursor-pointer items-center gap-1 rounded-sm text-left text-xs transition-colors focus-visible:ring-2 focus-visible:outline-none"
        >
          <span className="truncate">{models.slice(0, 2).join(" · ")}</span>
          {models.length > 2 ? <span className="shrink-0">+{models.length - 2}</span> : null}
          <ChevronDown
            className="size-3.5 shrink-0 group-data-[state=open]:rotate-180 motion-safe:transition-transform"
            aria-hidden="true"
          />
          <span className="sr-only">{t("providers.showModels")}</span>
        </button>
      </Popover.Trigger>
      {/* 浮层挂到卡片外，避免模型列表撑高当前卡片及同一行的相邻卡片。 */}
      <Popover.Portal>
        <Popover.Content
          side="bottom"
          align="start"
          sideOffset={8}
          collisionPadding={12}
          sticky="always"
          aria-labelledby={titleId}
          aria-describedby={undefined}
          className="bg-popover text-popover-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 motion-safe:data-[state=closed]:zoom-out-95 z-50 flex max-h-[min(320px,var(--radix-popover-content-available-height))] w-[min(320px,calc(100vw-1.5rem))] flex-col gap-3 overflow-hidden rounded-xl border p-3 shadow-xl outline-none motion-safe:duration-150"
        >
          <div className="flex shrink-0 items-center justify-between gap-2">
            <h4 id={titleId} className="min-w-0 text-xs font-medium break-words">
              {title}
            </h4>
            <Popover.Close asChild>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                aria-label={t("providers.hideModels")}
                className="shrink-0 cursor-pointer"
              >
                <X className="size-3.5" aria-hidden="true" />
              </Button>
            </Popover.Close>
          </div>
          <ScrollRegion as="section" ariaLabel={title} className="min-h-0 flex-1">
            <ul className="flex flex-wrap gap-1.5">
              {models.map((model) => (
                <li
                  key={model}
                  className="bg-muted/60 max-w-full rounded-md px-2 py-1 text-xs break-all select-text"
                >
                  {model}
                </li>
              ))}
            </ul>
          </ScrollRegion>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
