import { describe, expect, it } from "vitest"
import { formatAccountTime } from "./account-display"

describe("account timestamp", () => {
  it("uses a consistent compact local timestamp", () => {
    const timestamp = new Date(2026, 9, 8, 21, 12).toISOString()
    expect(formatAccountTime(timestamp)).toBe("2026-10-08 21:12")
    expect(formatAccountTime("invalid")).toBe("—")
  })
})
