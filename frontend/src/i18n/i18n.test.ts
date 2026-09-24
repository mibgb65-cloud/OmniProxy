import i18next from "i18next"
import { afterEach, describe, expect, it } from "vitest"
import { initI18n } from "./config"
import { en } from "./locales/en"
import { zh } from "./locales/zh"
import { defaultLanguage, isLanguage, languageOptions } from "./translation"

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

describe("i18next instance", () => {
  afterEach(async () => {
    await i18next.changeLanguage(defaultLanguage)
  })

  it("translates keys for the active language", async () => {
    await initI18n()
    expect(i18next.t("nav.dashboard")).toBe("仪表盘")

    await i18next.changeLanguage("en")
    expect(i18next.t("nav.dashboard")).toBe("Dashboard")
  })

  it("interpolates named variables", async () => {
    await i18next.changeLanguage("zh")
    expect(i18next.t("dashboard.pingReply", { result: "pong" })).toBe("后端回复：pong")
  })

  it("falls back to the key for unknown keys", () => {
    expect(i18next.t("nope.missing" as never)).toBe("nope.missing")
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
