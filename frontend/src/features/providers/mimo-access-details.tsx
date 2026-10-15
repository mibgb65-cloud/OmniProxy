import { useId } from "react"
import { ChevronDown, X } from "lucide-react"
import { Popover } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { ScrollRegion } from "../../components/common/scroll-region"
import { MIMO_PLAN_REGIONS, miMoEndpoint, miMoAnthropicBaseURL } from "../../services/mimo"

export function MiMoAccessDetails() {
  const { t } = useTranslation()
  const titleId = useId()
  const descriptionId = useId()
  return (
    <Popover.Root modal={false}>
      <Popover.Trigger asChild>
        <Button
          type="button"
          variant="outline"
          size="xs"
          aria-label={t("mimo.openAccess")}
          className="cursor-pointer gap-1 border-blue-200 bg-blue-50 font-normal text-blue-700 dark:border-blue-500/30 dark:bg-blue-500/10 dark:text-blue-300"
        >
          {t("mimo.plan")}
          <ChevronDown className="size-3" aria-hidden="true" />
        </Button>
      </Popover.Trigger>
      {/* 地址明细挂到卡片外，展开时不改变当前或相邻卡片的高度。 */}
      <Popover.Portal>
        <Popover.Content
          side="bottom"
          align="start"
          sideOffset={8}
          collisionPadding={12}
          sticky="always"
          aria-labelledby={titleId}
          aria-describedby={descriptionId}
          className="bg-popover text-popover-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 z-50 flex max-h-[min(480px,var(--radix-popover-content-available-height))] w-[min(420px,calc(100vw-1.5rem))] flex-col gap-3 overflow-hidden rounded-xl border p-4 shadow-xl outline-none motion-safe:duration-150"
        >
          <div className="flex shrink-0 items-center justify-between gap-2">
            <h4 id={titleId} className="text-sm font-semibold">
              {t("mimo.accessTitle")}
            </h4>
            <Popover.Close asChild>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                aria-label={t("mimo.closeAccess")}
                className="shrink-0 cursor-pointer"
              >
                <X className="size-3.5" aria-hidden="true" />
              </Button>
            </Popover.Close>
          </div>
          <ScrollRegion as="section" ariaLabel={t("mimo.accessTitle")} className="min-h-0 flex-1">
            <div className="space-y-3 text-xs leading-5">
              <p id={descriptionId} className="text-muted-foreground">
                {t("mimo.accessDescription")}
              </p>
              {MIMO_PLAN_REGIONS.map((region) => {
                const options = { region, billingMode: "token_plan", version: 0 }
                return (
                  <section key={region} className="space-y-1.5 rounded-lg border p-2.5">
                    <h5 className="font-medium">
                      {t(`mimo.${region}`)}{" "}
                      <span className="text-muted-foreground font-mono">{region}</span>
                    </h5>
                    <dl className="space-y-1.5">
                      <div>
                        <dt className="text-muted-foreground">OpenAI</dt>
                        <dd className="font-mono wrap-anywhere select-text">
                          {miMoEndpoint(options)}
                        </dd>
                      </div>
                      <div>
                        <dt className="text-muted-foreground">Anthropic</dt>
                        <dd className="font-mono wrap-anywhere select-text">
                          {miMoAnthropicBaseURL(options)}
                        </dd>
                      </div>
                    </dl>
                  </section>
                )
              })}
              <p className="text-muted-foreground">{t("mimo.accessModels")}</p>
              <p className="text-muted-foreground">{t("mimo.probeScope")}</p>
            </div>
          </ScrollRegion>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
