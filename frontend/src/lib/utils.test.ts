import { describe, expect, it } from "vitest"
import { cn } from "./utils"

describe("cn", () => {
  it("merges truthy class names and skips falsy ones", () => {
    const hidden = false
    expect(cn("px-2", hidden && "hidden", undefined, "py-1")).toBe("px-2 py-1")
  })

  it("lets later Tailwind utilities win", () => {
    // tailwind-merge keeps the position of the first occurrence.
    expect(cn("px-2 py-1", "px-4")).toBe("py-1 px-4")
  })

  it("returns an empty string for no input", () => {
    expect(cn()).toBe("")
  })
})
