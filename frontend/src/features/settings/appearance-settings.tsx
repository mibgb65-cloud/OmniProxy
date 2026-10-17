import { Slider } from "radix-ui"
import { Type } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent } from "../../components/ui/card"
import { defaultFontScale, fontScaleStep, maxFontScale, minFontScale } from "../../lib/typography"
import { useUIStore } from "../../stores/ui-store"

export function AppearanceSettings() {
  const { t } = useTranslation()
  const fontScale = useUIStore((state) => state.fontScale)
  const setFontScale = useUIStore((state) => state.setFontScale)
  return (
    <Card className="gap-0 rounded-2xl py-4 shadow-none">
      <CardContent className="space-y-3 px-5">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="min-w-0 space-y-1">
            <div className="flex flex-wrap items-center gap-2 text-sm">
              <Type className="size-4" aria-hidden="true" />
              <span className="font-medium">{t("settings.appearance.fontSize")}</span>
              <output className="text-muted-foreground font-mono tabular-nums">{fontScale}%</output>
            </div>
            <p className="text-muted-foreground text-xs">{t("settings.appearance.description")}</p>
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={fontScale === defaultFontScale}
            onClick={() => setFontScale(defaultFontScale)}
          >
            {t("settings.appearance.reset")}
          </Button>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-muted-foreground text-xs tabular-nums">{minFontScale}%</span>
          <Slider.Root
            min={minFontScale}
            max={maxFontScale}
            step={fontScaleStep}
            value={[fontScale]}
            onValueChange={(values) => {
              if (values[0] !== undefined) setFontScale(values[0])
            }}
            className="relative flex h-6 w-full touch-none items-center select-none"
          >
            <Slider.Track className="bg-muted relative h-2 w-full grow overflow-hidden rounded-full">
              <Slider.Range className="bg-primary absolute h-full" />
            </Slider.Track>
            <Slider.Thumb
              aria-label={t("settings.appearance.fontSize")}
              aria-valuetext={`${fontScale}%`}
              className="border-primary bg-background focus-visible:ring-ring/50 block size-5 rounded-full border-2 shadow-sm outline-none focus-visible:ring-4"
            />
          </Slider.Root>
          <span className="text-muted-foreground text-xs tabular-nums">{maxFontScale}%</span>
        </div>
        <div className="bg-muted/40 space-y-1 rounded-lg px-3 py-2.5">
          <p className="text-sm">{t("settings.appearance.preview")}</p>
          <p className="text-muted-foreground text-xs">
            Aa 0123456789 · {t("settings.appearance.previewDetail")}
          </p>
        </div>
      </CardContent>
    </Card>
  )
}
