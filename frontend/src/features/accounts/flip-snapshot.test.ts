import { describe, expect, it, vi } from "vitest"
import { takeFlipSnapshot } from "./flip-snapshot"

describe("takeFlipSnapshot", () => {
  it("preserves geometry without duplicating DOM identity or replaying CSS animations", () => {
    const source = document.createElement("div")
    source.id = "live-provider"
    source.dataset.flipId = "provider"
    source.dataset.flipFlying = "true"
    const input = document.createElement("input")
    input.id = "live-input"
    input.autofocus = true
    source.append(input)
    source.getBoundingClientRect = () => new DOMRect(20, 40, 400, 320)
    const snapshot = takeFlipSnapshot(source)
    expect(snapshot.rect).toEqual(new DOMRect(20, 40, 400, 320))
    expect(snapshot.content.style.width).toBe("400px")
    expect(snapshot.content.style.height).toBe("320px")
    expect(snapshot.content.id).toBe("")
    expect(snapshot.content.dataset.flipId).toBeUndefined()
    expect(snapshot.content.dataset.flipFlying).toBeUndefined()
    expect(snapshot.content.querySelector("[id], [autofocus]")).toBeNull()
    expect(snapshot.content.style.animation).toBe("none")
    expect(snapshot.content.firstElementChild).toHaveStyle({
      animation: "none",
      transition: "none",
    })
    expect(source.id).toBe("live-provider")
    expect(input.autofocus).toBe(true)
  })

  it("freezes the displayed opacity and transform of an interrupted layer", () => {
    const source = document.createElement("div")
    const layer = document.createElement("div")
    layer.style.opacity = "0.4"
    layer.style.transform = "scale(2)"
    Object.defineProperty(layer, "getAnimations", { value: vi.fn(() => [{}]) })
    source.append(layer)
    const snapshot = takeFlipSnapshot(source)
    expect(snapshot.content.firstElementChild).toHaveStyle({
      opacity: "0.4",
      transform: "scale(2)",
      animation: "none",
    })
  })
})
