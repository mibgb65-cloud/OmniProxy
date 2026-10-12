import { beforeEach, describe, expect, it } from "vitest"
import { defaultLanguage } from "../i18n/translation"
import { defaultTheme } from "../lib/theme"
import { useUIStore } from "./ui-store"

describe("UI theme preference", () => {
  beforeEach(() => {
    localStorage.clear()
    useUIStore.setState({
      language: defaultLanguage,
      theme: defaultTheme,
      accountView: "cards",
      fontScale: 100,
    })
  })

  it("restores a saved light theme", async () => {
    useUIStore.getState().toggleTheme()
    expect(useUIStore.getState().theme).toBe("light")

    useUIStore.setState({ theme: defaultTheme })
    localStorage.setItem(
      "omniproxy-ui-state",
      JSON.stringify({ state: { theme: "light", language: "zh" }, version: 0 }),
    )
    await useUIStore.persist.rehydrate()

    expect(useUIStore.getState().theme).toBe("light")
  })

  it("rejects an unknown stored theme", async () => {
    localStorage.setItem(
      "omniproxy-ui-state",
      JSON.stringify({
        state: {
          theme: "unknown",
          language: "zh",
          sidebarCollapsed: "invalid",
          toggleTheme: "invalid",
        },
        version: 0,
      }),
    )
    await useUIStore.persist.rehydrate()

    expect(useUIStore.getState().theme).toBe(defaultTheme)
    expect(useUIStore.getState().sidebarCollapsed).toBe(false)
    expect(useUIStore.getState().toggleTheme).toBeTypeOf("function")
  })

  it("persists and restores the account list preference", async () => {
    useUIStore.getState().setAccountView("list")
    expect(JSON.parse(localStorage.getItem("omniproxy-ui-state")!).state.accountView).toBe("list")
    useUIStore.setState({ accountView: "cards" })
    localStorage.setItem(
      "omniproxy-ui-state",
      JSON.stringify({ state: { accountView: "list" }, version: 0 }),
    )
    await useUIStore.persist.rehydrate()
    expect(useUIStore.getState().accountView).toBe("list")
  })

  it("rejects invalid account view data without replacing its setter", async () => {
    localStorage.setItem(
      "omniproxy-ui-state",
      JSON.stringify({ state: { accountView: "unknown", setAccountView: "invalid" }, version: 0 }),
    )
    await useUIStore.persist.rehydrate()
    expect(useUIStore.getState().accountView).toBe("cards")
    expect(useUIStore.getState().setAccountView).toBeTypeOf("function")
  })

  it("persists font size and restores it without replacing the setter", async () => {
    useUIStore.getState().setFontScale(125)
    expect(JSON.parse(localStorage.getItem("omniproxy-ui-state")!).state.fontScale).toBe(125)
    useUIStore.setState({ fontScale: 100 })
    localStorage.setItem(
      "omniproxy-ui-state",
      JSON.stringify({ state: { fontScale: 125, setFontScale: "invalid" }, version: 0 }),
    )
    await useUIStore.persist.rehydrate()
    expect(useUIStore.getState().fontScale).toBe(125)
    expect(useUIStore.getState().setFontScale).toBeTypeOf("function")
  })

  it("rejects invalid font sizes in stored preferences and setters", async () => {
    for (const value of ["120", 0, 79, 141, 113, null]) {
      localStorage.setItem(
        "omniproxy-ui-state",
        JSON.stringify({ state: { fontScale: value }, version: 0 }),
      )
      await useUIStore.persist.rehydrate()
      expect(useUIStore.getState().fontScale).toBe(100)
    }
    useUIStore.getState().setFontScale(120)
    for (const value of [NaN, Infinity, 200, 85.5]) useUIStore.getState().setFontScale(value)
    expect(useUIStore.getState().fontScale).toBe(120)
  })
})
