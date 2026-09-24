import { describe, expect, it } from "vitest"
import { defaultLanguage, isLanguage, languageOptions, translate } from "./translation"
import { en } from "./locales/en"
import { zh } from "./locales/zh"

describe("translate", () => {
  it("returns the string for the selected language", () => {
    expect(translate("zh", "nav.dashboard")).toBe("仪表盘")
    expect(translate("en", "nav.dashboard")).toBe("Dashboard")
  })

  it("substitutes named variables", () => {
    expect(translate("zh", "dashboard.pingReply", { result: "pong" })).toBe("后端回复：pong")
    expect(translate("en", "dashboard.pingReply", { result: "pong" })).toBe("Backend replied: pong")
  })

  it("keeps placeholders that have no matching variable", () => {
    expect(translate("en", "dashboard.loadFailed", {})).toBe("Failed to load app info: {error}")
  })

  it("falls back to the key itself for unknown keys", () => {
    // Cast exercises the runtime fallback; the type system prevents this.
    expect(translate("en", "nope.missing" as never)).toBe("nope.missing")
  })
})

describe("dictionaries", () => {
  it("cover exactly the same keys", () => {
    expect(Object.keys(en).sort()).toEqual(Object.keys(zh).sort())
  })

  it("have no empty strings", () => {
    for (const [key, value] of Object.entries({ ...zh, ...en })) {
      expect(value.trim(), `empty value for ${key}`).not.toBe("")
    }
  })
})

describe("language metadata", () => {
  it("exposes both languages as options", () => {
    expect(languageOptions.map((o) => o.value)).toEqual(["zh", "en"])
  })

  it("defaults to Chinese", () => {
    expect(defaultLanguage).toBe("zh")
  })

  it("guards language values", () => {
    expect(isLanguage("zh")).toBe(true)
    expect(isLanguage("en")).toBe(true)
    expect(isLanguage("fr")).toBe(false)
    expect(isLanguage(undefined)).toBe(false)
  })
})
