import { describe, expect, it } from "vitest"
import { formatBytes } from "./format"

describe("formatBytes", () => {
  it("formats byte ranges with the right unit and precision", () => {
    expect(formatBytes(0)).toBe("0 B")
    expect(formatBytes(-5)).toBe("0 B")
    expect(formatBytes(512)).toBe("512 B")
    expect(formatBytes(1024)).toBe("1.0 KB")
    expect(formatBytes(1536)).toBe("1.5 KB")
    expect(formatBytes(1024 * 1024 * 5)).toBe("5.0 MB")
    expect(formatBytes(1024 * 1024 * 1024 * 2)).toBe("2.0 GB")
  })

  it("drops decimals for larger values", () => {
    expect(formatBytes(1024 * 100)).toBe("100 KB")
    expect(formatBytes(1024 * 1024 * 512)).toBe("512 MB")
  })

  it("treats non-finite input as zero", () => {
    expect(formatBytes(Number.NaN)).toBe("0 B")
    expect(formatBytes(Number.POSITIVE_INFINITY)).toBe("0 B")
  })
})
