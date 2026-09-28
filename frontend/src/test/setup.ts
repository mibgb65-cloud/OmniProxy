import "@testing-library/jest-dom/vitest"
import { cleanup } from "@testing-library/react"
import { afterEach } from "vitest"
import { initI18n } from "../i18n/config"

// Initialize the shared i18next instance before any component renders, so
// `useTranslation` returns real translations instead of raw keys.
await initI18n()

// jsdom 不计算布局，也没有 ResizeObserver；滚动尺寸由相关组件测试单独模拟。
class ResizeObserverStub {
  observe() {}
  disconnect() {}
  unobserve() {}
}

globalThis.ResizeObserver = ResizeObserverStub

afterEach(() => {
  cleanup()
})
