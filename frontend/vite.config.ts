import { defineConfig } from "vitest/config"
import react from "@vitejs/plugin-react"
import wails from "@wailsio/runtime/plugins/vite"
import tailwindcss from "@tailwindcss/vite"
import { unpluginRouterGeneratorFactory } from "@tanstack/router-plugin"
import { fileURLToPath } from "node:url"

// File-based routing: generates src/app/routeTree.gen.ts from src/app/routes.
const routerGenerator = unpluginRouterGeneratorFactory({
  target: "react",
  routesDirectory: "./src/app/routes",
  generatedRouteTree: "./src/app/routeTree.gen.ts",
})

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [routerGenerator.vite, react(), tailwindcss(), wails("./bindings")],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  test: {
    environment: "jsdom",
    globals: false,
    setupFiles: ["./src/test/setup.ts"],
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
  },
})
