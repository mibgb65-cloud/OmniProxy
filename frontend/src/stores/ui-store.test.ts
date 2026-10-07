import { beforeEach, describe, expect, it } from "vitest"
import { defaultLanguage } from "../i18n/translation"
import { defaultTheme } from "../lib/theme"
import { useUIStore } from "./ui-store"

describe("UI theme preference", () => {
  beforeEach(() => {
    localStorage.clear()
    useUIStore.setState({ language: defaultLanguage, theme: defaultTheme, accountView: "cards" })
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
})
