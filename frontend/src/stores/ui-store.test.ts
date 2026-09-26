import { beforeEach, describe, expect, it } from "vitest"
import { defaultLanguage } from "../i18n/translation"
import { defaultTheme } from "../lib/theme"
import { useUIStore } from "./ui-store"

describe("UI theme preference", () => {
  beforeEach(() => {
    localStorage.clear()
    useUIStore.setState({ language: defaultLanguage, theme: defaultTheme })
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
})
