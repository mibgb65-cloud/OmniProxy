export const defaultFontScale = 100
export const minFontScale = 80
export const maxFontScale = 140
export const fontScaleStep = 5

export function isFontScale(value: unknown): value is number {
  return (
    typeof value === "number" &&
    Number.isFinite(value) &&
    value >= minFontScale &&
    value <= maxFontScale &&
    value % fontScaleStep === 0
  )
}

export function applyFontScale(value: number): void {
  document.documentElement.style.fontSize = `${isFontScale(value) ? value : defaultFontScale}%`
}
