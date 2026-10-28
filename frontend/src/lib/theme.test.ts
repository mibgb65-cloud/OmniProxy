import { describe, expect, it } from "vitest"
import { themeRevealRadius } from "./theme"

describe("themeRevealRadius", () => {
  it("uses the farthest corner distance so the circle always covers the window", () => {
    expect(themeRevealRadius(100, 50, 800, 400)).toBeCloseTo(Math.hypot(700, 350))
    expect(themeRevealRadius(0, 0, 800, 400)).toBeCloseTo(Math.hypot(800, 400))
    expect(themeRevealRadius(800, 400, 800, 400)).toBeCloseTo(Math.hypot(800, 400))
    expect(themeRevealRadius(400, 200, 800, 400)).toBeCloseTo(Math.hypot(400, 200))
  })
})
